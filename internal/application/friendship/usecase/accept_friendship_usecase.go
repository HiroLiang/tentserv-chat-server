package usecase

import (
	"context"
	"errors"
	"strings"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"go.uber.org/zap"
)

type AcceptFriendshipInput struct {
	FriendshipID int64
}

type AcceptFriendshipUseCase struct {
	uow             transaction.UnitOfWork
	friendshipRepo  friendship.Repository
	participantRepo participant.Repository
	chatRoomRepo    chatroom.Repository
	chatMemberRepo  chatmember.Repository
}

func NewAcceptFriendshipUseCase(
	uow transaction.UnitOfWork,
	friendshipRepo friendship.Repository,
	participantRepo participant.Repository,
	chatRoomRepo chatroom.Repository,
	chatMemberRepo chatmember.Repository,
) *AcceptFriendshipUseCase {
	return &AcceptFriendshipUseCase{
		uow:             uow,
		friendshipRepo:  friendshipRepo,
		participantRepo: participantRepo,
		chatRoomRepo:    chatRoomRepo,
		chatMemberRepo:  chatMemberRepo,
	}
}

func (uc *AcceptFriendshipUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[AcceptFriendshipInput],
) error {
	currentUserID := input.Base.Auth.UserID

	ctx, tx, err := uc.uow.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// FindByID inside transaction to prevent concurrent double-accept
	f, err := uc.friendshipRepo.FindByID(ctx, input.Data.FriendshipID)
	if err != nil {
		if errors.Is(err, friendship.ErrFriendshipNotFound) {
			return friendship.ErrFriendshipNotFound
		}
		return err
	}

	if f.FriendID != currentUserID {
		return friendship.ErrForbidden
	}

	if f.Status != friendship.StatusPending {
		return friendship.ErrFriendshipNotPending
	}

	if err := uc.friendshipRepo.UpdateStatus(ctx, f.ID, friendship.StatusAccepted); err != nil {
		return err
	}

	// Create the reverse friendship record (accepter→requester)
	if err := uc.friendshipRepo.Create(ctx, currentUserID, f.UserID); err != nil {
		// Concurrent accept: duplicate key means reverse already exists — treat as success
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate key value") {
			return err
		}
	}

	// Update the reverse row status to accepted
	reverse, err := uc.friendshipRepo.FindByUserIDAndFriendID(ctx, currentUserID, f.UserID)
	if err != nil {
		return err
	}
	if err := uc.friendshipRepo.UpdateStatus(ctx, reverse.ID, friendship.StatusAccepted); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Auto-create a DIRECT room for the two friends (best-effort, outside transaction)
	accepterP, err := uc.participantRepo.FindByUserID(context.Background(), currentUserID)
	if err != nil {
		logger.Log.Warn("accept friendship: accepter participant not found, skipping direct room creation",
			zap.Any("userID", currentUserID), zap.Error(err))
		return nil
	}
	requesterP, err := uc.participantRepo.FindByUserID(context.Background(), f.UserID)
	if err != nil {
		logger.Log.Warn("accept friendship: requester participant not found, skipping direct room creation",
			zap.Any("userID", f.UserID), zap.Error(err))
		return nil
	}

	_, err = uc.chatRoomRepo.FindDirectByParticipants(context.Background(), accepterP.ID, requesterP.ID)
	if errors.Is(err, chatroom.ErrNotFound) {
		room := &chatroom.ChatRoom{Type: chatroom.Direct, MaxMembers: 2}
		if createErr := uc.chatRoomRepo.Create(context.Background(), room); createErr != nil {
			logger.Log.Warn("accept friendship: failed to create direct chat room", zap.Error(createErr))
			return nil
		}
		if addErr := uc.chatMemberRepo.Add(context.Background(), &chatmember.ChatMember{
			RoomID:        room.ID,
			ParticipantID: accepterP.ID,
			Role:          chatmember.Member,
		}); addErr != nil {
			logger.Log.Warn("accept friendship: failed to add accepter to direct room", zap.Error(addErr))
		}
		if addErr := uc.chatMemberRepo.Add(context.Background(), &chatmember.ChatMember{
			RoomID:        room.ID,
			ParticipantID: requesterP.ID,
			Role:          chatmember.Member,
		}); addErr != nil {
			logger.Log.Warn("accept friendship: failed to add requester to direct room", zap.Error(addErr))
		}
	}

	return nil
}

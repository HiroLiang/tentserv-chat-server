package usecase

import (
	"context"
	"errors"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
)

type RemoveFriendshipInput struct {
	FriendshipID int64
}

type RemovedDirectRoomInfo struct {
	RoomID    int64
	MemberIDs []int64
}

type RemoveFriendshipOutput struct {
	DeletedDirectRoom *RemovedDirectRoomInfo
}

type RemoveFriendshipUseCase struct {
	uow             transaction.UnitOfWork
	friendshipRepo  friendship.Repository
	participantRepo participant.Repository
	chatRoomRepo    chatroom.Repository
	chatMemberRepo  chatmember.Repository
}

func NewRemoveFriendshipUseCase(
	uow transaction.UnitOfWork,
	friendshipRepo friendship.Repository,
	participantRepo participant.Repository,
	chatRoomRepo chatroom.Repository,
	chatMemberRepo chatmember.Repository,
) *RemoveFriendshipUseCase {
	return &RemoveFriendshipUseCase{
		uow:             uow,
		friendshipRepo:  friendshipRepo,
		participantRepo: participantRepo,
		chatRoomRepo:    chatRoomRepo,
		chatMemberRepo:  chatMemberRepo,
	}
}

func (uc *RemoveFriendshipUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[RemoveFriendshipInput],
) (*RemoveFriendshipOutput, error) {
	currentUserID := input.Base.Auth.UserID

	ctx, tx, err := uc.uow.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	f, err := uc.friendshipRepo.FindByID(ctx, input.Data.FriendshipID)
	if err != nil {
		if errors.Is(err, friendship.ErrFriendshipNotFound) {
			return nil, friendship.ErrFriendshipNotFound
		}
		return nil, err
	}

	if f.UserID != currentUserID && f.FriendID != currentUserID {
		return nil, friendship.ErrForbidden
	}

	// Delete the primary record
	if err := uc.friendshipRepo.Delete(ctx, f.ID); err != nil {
		return nil, err
	}

	// Delete the reverse record (created on acceptance) to keep data consistent
	if reverse, err := uc.friendshipRepo.FindByUserIDAndFriendID(ctx, f.FriendID, f.UserID); err == nil {
		if err := uc.friendshipRepo.Delete(ctx, reverse.ID); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, friendship.ErrFriendshipNotFound) {
		return nil, err
	}

	out := &RemoveFriendshipOutput{}
	if f.Status == friendship.StatusAccepted {
		deletedRoom, err := uc.softDeleteAcceptedDirectRoom(ctx, f.UserID, f.FriendID)
		if err != nil {
			return nil, err
		}
		out.DeletedDirectRoom = deletedRoom
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return out, nil
}

func (uc *RemoveFriendshipUseCase) softDeleteAcceptedDirectRoom(
	ctx context.Context,
	userID1 shared.UserID,
	userID2 shared.UserID,
) (*RemovedDirectRoomInfo, error) {
	p1, err := uc.participantRepo.FindByUserID(ctx, userID1)
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	p2, err := uc.participantRepo.FindByUserID(ctx, userID2)
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	room, err := uc.chatRoomRepo.FindDirectByParticipants(ctx, p1.ID, p2.ID)
	if err != nil {
		if errors.Is(err, chatroom.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	members, err := uc.chatMemberRepo.FindByRoom(ctx, room.ID)
	if err != nil {
		return nil, err
	}

	memberIDs := make([]int64, 0, 2)
	for _, member := range members {
		if member.IsDeleted {
			continue
		}
		if member.ParticipantID != p1.ID && member.ParticipantID != p2.ID {
			continue
		}
		if err := uc.chatMemberRepo.SoftDelete(ctx, member.ID); err != nil {
			return nil, err
		}
		memberIDs = append(memberIDs, int64(member.ID))
	}

	if err := uc.chatRoomRepo.SoftDelete(ctx, room.ID); err != nil {
		return nil, err
	}

	return &RemovedDirectRoomInfo{
		RoomID:    int64(room.ID),
		MemberIDs: memberIDs,
	}, nil
}

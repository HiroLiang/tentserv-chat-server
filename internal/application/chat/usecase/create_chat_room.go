package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatinvitation"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
)

type CreateChatRoomInput struct {
	Name        string
	Description string
	Type        chatroom.RoomType
	MaxMembers  int
	AllowAgent  bool
	MemberIDs   []int64
}

type CreateChatRoomOutput struct {
	ID             int64
	Name           string
	Type           string
	MaxMembers     int
	AllowAgent     bool
	CreatedAt      time.Time
	AlreadyExisted bool
}

type CreateChatRoomUseCase struct {
	uow             transaction.UnitOfWork
	chatroomRepo    chatroom.Repository
	chatMemberRepo  chatmember.Repository
	participantRepo participant.Repository
	friendshipRepo  friendship.Repository
	invitationRepo  chatinvitation.Repository
}

func NewCreateChatRoomUseCase(
	uow transaction.UnitOfWork,
	chatroomRepo chatroom.Repository,
	chatMemberRepo chatmember.Repository,
	participantRepo participant.Repository,
	friendshipRepo friendship.Repository,
	invitationRepo chatinvitation.Repository,
) *CreateChatRoomUseCase {
	return &CreateChatRoomUseCase{
		uow:             uow,
		chatroomRepo:    chatroomRepo,
		chatMemberRepo:  chatMemberRepo,
		participantRepo: participantRepo,
		friendshipRepo:  friendshipRepo,
		invitationRepo:  invitationRepo,
	}
}

func (uc *CreateChatRoomUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[CreateChatRoomInput],
) (CreateChatRoomOutput, error) {
	userID := input.Base.Auth.UserID

	ctx, tx, err := uc.uow.Begin(ctx)
	if err != nil {
		return CreateChatRoomOutput{}, ErrChatRoomCreate
	}
	defer func() { _ = tx.Rollback() }()

	p, err := uc.participantRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return CreateChatRoomOutput{}, ErrParticipantNotFound
		}
		return CreateChatRoomOutput{}, ErrChatRoomCreate
	}

	// DIRECT room with one invited member: check for existing room, blocking, then invite
	isDirect := strings.EqualFold(string(input.Data.Type), string(chatroom.Direct))
	if isDirect && len(input.Data.MemberIDs) == 1 {
		invitedUserID := shared.UserID(input.Data.MemberIDs[0])

		invitedP, err := uc.participantRepo.FindByUserID(ctx, invitedUserID)
		if err != nil {
			if errors.Is(err, participant.ErrNotFound) {
				return CreateChatRoomOutput{}, ErrParticipantNotFound
			}
			return CreateChatRoomOutput{}, ErrChatRoomCreate
		}

		fs, err := uc.friendshipRepo.FindBetweenUsers(ctx, userID, invitedUserID)
		if err != nil && !errors.Is(err, friendship.ErrFriendshipNotFound) {
			return CreateChatRoomOutput{}, ErrChatRoomCreate
		}
		if fs != nil && fs.Status == friendship.StatusBlocked {
			return CreateChatRoomOutput{}, ErrUserBlocked
		}

		existing, err := uc.chatroomRepo.FindDirectByParticipants(ctx, p.ID, invitedP.ID)
		if err != nil && !errors.Is(err, chatroom.ErrNotFound) {
			return CreateChatRoomOutput{}, ErrChatRoomCreate
		}
		if existing != nil {
			return CreateChatRoomOutput{
				ID:             int64(existing.ID),
				Name:           existing.Name,
				Type:           string(existing.Type),
				MaxMembers:     existing.MaxMembers,
				AllowAgent:     existing.AllowAgent,
				CreatedAt:      existing.CreatedAt,
				AlreadyExisted: true,
			}, nil
		}

		room := &chatroom.ChatRoom{
			Name:        input.Data.Name,
			Description: input.Data.Description,
			Type:        input.Data.Type,
			MaxMembers:  input.Data.MaxMembers,
			AllowAgent:  input.Data.AllowAgent,
		}
		if err := uc.chatroomRepo.Create(ctx, room); err != nil {
			return CreateChatRoomOutput{}, ErrChatRoomCreate
		}

		member := &chatmember.ChatMember{
			RoomID:        room.ID,
			ParticipantID: p.ID,
			Role:          chatmember.Owner,
		}
		if err := uc.chatMemberRepo.Add(ctx, member); err != nil {
			return CreateChatRoomOutput{}, ErrChatRoomCreate
		}

		inv := &chatinvitation.ChatInvitation{
			RoomID:         room.ID,
			InviterID:      p.ID,
			InviteeID:      invitedP.ID,
			Status:         chatinvitation.Pending,
			InvitationType: chatinvitation.Invitation,
		}
		if err := uc.invitationRepo.Create(ctx, inv); err != nil {
			return CreateChatRoomOutput{}, ErrInvitationCreate
		}

		if err := tx.Commit(); err != nil {
			return CreateChatRoomOutput{}, ErrChatRoomCreate
		}

		return CreateChatRoomOutput{
			ID:         int64(room.ID),
			Name:       room.Name,
			Type:       string(room.Type),
			MaxMembers: room.MaxMembers,
			AllowAgent: room.AllowAgent,
			CreatedAt:  room.CreatedAt,
		}, nil
	}

	room := &chatroom.ChatRoom{
		Name:        input.Data.Name,
		Description: input.Data.Description,
		Type:        input.Data.Type,
		MaxMembers:  input.Data.MaxMembers,
		AllowAgent:  input.Data.AllowAgent,
	}
	if err := uc.chatroomRepo.Create(ctx, room); err != nil {
		return CreateChatRoomOutput{}, ErrChatRoomCreate
	}

	member := &chatmember.ChatMember{
		RoomID:        room.ID,
		ParticipantID: p.ID,
		Role:          chatmember.Owner,
	}
	if err := uc.chatMemberRepo.Add(ctx, member); err != nil {
		return CreateChatRoomOutput{}, ErrChatRoomCreate
	}

	for _, memberID := range input.Data.MemberIDs {
		invitedP, err := uc.participantRepo.FindByUserID(ctx, shared.UserID(memberID))
		if err != nil {
			if errors.Is(err, participant.ErrNotFound) {
				return CreateChatRoomOutput{}, ErrParticipantNotFound
			}
			return CreateChatRoomOutput{}, ErrChatRoomCreate
		}
		inv := &chatinvitation.ChatInvitation{
			RoomID:         room.ID,
			InviterID:      p.ID,
			InviteeID:      invitedP.ID,
			Status:         chatinvitation.Pending,
			InvitationType: chatinvitation.Invitation,
		}
		if err := uc.invitationRepo.Create(ctx, inv); err != nil {
			return CreateChatRoomOutput{}, ErrInvitationCreate
		}
	}

	if err := tx.Commit(); err != nil {
		return CreateChatRoomOutput{}, ErrChatRoomCreate
	}

	return CreateChatRoomOutput{
		ID:         int64(room.ID),
		Name:       room.Name,
		Type:       string(room.Type),
		MaxMembers: room.MaxMembers,
		AllowAgent: room.AllowAgent,
		CreatedAt:  room.CreatedAt,
	}, nil
}

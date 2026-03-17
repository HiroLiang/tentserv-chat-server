package usecase

import (
	"context"
	"errors"
	"time"

	appShared "github.com/HiroLiang/goat-server/internal/application/shared"
	"github.com/HiroLiang/goat-server/internal/domain/chatmember"
	"github.com/HiroLiang/goat-server/internal/domain/chatroom"
	"github.com/HiroLiang/goat-server/internal/domain/participant"
	"github.com/HiroLiang/goat-server/internal/domain/transaction"
)

type CreateChatRoomInput struct {
	Name        string
	Description string
	Type        chatroom.RoomType
	MaxMembers  int
	AllowAgent  bool
}

type CreateChatRoomOutput struct {
	ID         int64
	Name       string
	Type       string
	MaxMembers int
	AllowAgent bool
	CreatedAt  time.Time
}

type CreateChatRoomUseCase struct {
	uow             transaction.UnitOfWork
	chatroomRepo    chatroom.Repository
	chatMemberRepo  chatmember.Repository
	participantRepo participant.Repository
}

func NewCreateChatRoomUseCase(
	uow transaction.UnitOfWork,
	chatroomRepo chatroom.Repository,
	chatMemberRepo chatmember.Repository,
	participantRepo participant.Repository,
) *CreateChatRoomUseCase {
	return &CreateChatRoomUseCase{
		uow:             uow,
		chatroomRepo:    chatroomRepo,
		chatMemberRepo:  chatMemberRepo,
		participantRepo: participantRepo,
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

package usecase

import (
	"context"
	"errors"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatinvitation"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
)

type JoinChatRoomInput struct {
	RoomID int64
}

type JoinChatRoomOutput struct {
	MemberID     *int64
	Role         *string
	JoinedAt     *time.Time
	InvitationID *int64
	Status       *string
}

type JoinChatRoomUseCase struct {
	uow             transaction.UnitOfWork
	chatroomRepo    chatroom.Repository
	chatMemberRepo  chatmember.Repository
	participantRepo participant.Repository
	invitationRepo  chatinvitation.Repository
}

func NewJoinChatRoomUseCase(
	uow transaction.UnitOfWork,
	chatroomRepo chatroom.Repository,
	chatMemberRepo chatmember.Repository,
	participantRepo participant.Repository,
	invitationRepo chatinvitation.Repository,
) *JoinChatRoomUseCase {
	return &JoinChatRoomUseCase{
		uow:             uow,
		chatroomRepo:    chatroomRepo,
		chatMemberRepo:  chatMemberRepo,
		participantRepo: participantRepo,
		invitationRepo:  invitationRepo,
	}
}

func (uc *JoinChatRoomUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[JoinChatRoomInput],
) (JoinChatRoomOutput, error) {
	userID := input.Base.Auth.UserID
	roomID := chatroom.ID(input.Data.RoomID)

	ctx, tx, err := uc.uow.Begin(ctx)
	if err != nil {
		return JoinChatRoomOutput{}, ErrChatRoomCreate
	}
	defer func() { _ = tx.Rollback() }()

	room, err := uc.chatroomRepo.FindByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, chatroom.ErrNotFound) {
			return JoinChatRoomOutput{}, ErrChatRoomNotFound
		}
		return JoinChatRoomOutput{}, ErrChatRoomCreate
	}
	_ = room

	p, err := uc.participantRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return JoinChatRoomOutput{}, ErrParticipantNotFound
		}
		return JoinChatRoomOutput{}, ErrChatRoomCreate
	}

	existing, err := uc.chatMemberRepo.FindByRoomAndParticipant(ctx, roomID, p.ID)
	if err != nil && !errors.Is(err, chatmember.ErrNotFound) {
		return JoinChatRoomOutput{}, ErrChatRoomCreate
	}
	if existing != nil && !existing.IsDeleted {
		return JoinChatRoomOutput{}, ErrAlreadyMember
	}

	if p.Type == participant.AgentType || p.Type == participant.SystemType {
		member := &chatmember.ChatMember{
			RoomID:        roomID,
			ParticipantID: p.ID,
			Role:          chatmember.Member,
		}
		if err := uc.chatMemberRepo.Add(ctx, member); err != nil {
			return JoinChatRoomOutput{}, ErrChatRoomCreate
		}
		if err := tx.Commit(); err != nil {
			return JoinChatRoomOutput{}, ErrChatRoomCreate
		}
		memberID := int64(member.ID)
		role := string(member.Role)
		return JoinChatRoomOutput{
			MemberID: &memberID,
			Role:     &role,
			JoinedAt: &member.JoinedAt,
		}, nil
	}

	_, err = uc.invitationRepo.FindByRoomAndInvitee(ctx, roomID, p.ID)
	if err == nil {
		return JoinChatRoomOutput{}, ErrInvitationAlreadyExists
	}
	if !errors.Is(err, chatinvitation.ErrNotFound) {
		return JoinChatRoomOutput{}, ErrInvitationCreate
	}

	inv := &chatinvitation.ChatInvitation{
		RoomID:    roomID,
		InviterID: p.ID,
		InviteeID: p.ID,
		Status:    chatinvitation.Pending,
	}
	if err := uc.invitationRepo.Create(ctx, inv); err != nil {
		return JoinChatRoomOutput{}, ErrInvitationCreate
	}

	if err := tx.Commit(); err != nil {
		return JoinChatRoomOutput{}, ErrInvitationCreate
	}

	invID := int64(inv.ID)
	status := string(inv.Status)
	return JoinChatRoomOutput{
		InvitationID: &invID,
		Status:       &status,
	}, nil
}

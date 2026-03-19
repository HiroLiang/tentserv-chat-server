package usecase

import (
	"context"
	"errors"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatinvitation"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
)

type ApproveJoinRequestInput struct {
	InvitationID int64
	Approve      bool
}

type ApproveJoinRequestOutput struct {
	InvitationID int64
	Status       string
	MemberID     *int64
	Role         *string
	JoinedAt     *time.Time
}

type ApproveJoinRequestUseCase struct {
	uow             transaction.UnitOfWork
	chatMemberRepo  chatmember.Repository
	participantRepo participant.Repository
	invitationRepo  chatinvitation.Repository
}

func NewApproveJoinRequestUseCase(
	uow transaction.UnitOfWork,
	chatMemberRepo chatmember.Repository,
	participantRepo participant.Repository,
	invitationRepo chatinvitation.Repository,
) *ApproveJoinRequestUseCase {
	return &ApproveJoinRequestUseCase{
		uow:             uow,
		chatMemberRepo:  chatMemberRepo,
		participantRepo: participantRepo,
		invitationRepo:  invitationRepo,
	}
}

func (uc *ApproveJoinRequestUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[ApproveJoinRequestInput],
) (ApproveJoinRequestOutput, error) {
	userID := input.Base.Auth.UserID
	invitationID := chatinvitation.ID(input.Data.InvitationID)

	ctx, tx, err := uc.uow.Begin(ctx)
	if err != nil {
		return ApproveJoinRequestOutput{}, ErrInvitationCreate
	}
	defer func() { _ = tx.Rollback() }()

	inv, err := uc.invitationRepo.FindByID(ctx, invitationID)
	if err != nil {
		if errors.Is(err, chatinvitation.ErrNotFound) {
			return ApproveJoinRequestOutput{}, ErrInvitationNotFound
		}
		return ApproveJoinRequestOutput{}, ErrInvitationCreate
	}

	if inv.Status != chatinvitation.Pending {
		return ApproveJoinRequestOutput{}, ErrInvitationAlreadyResolved
	}

	callerParticipant, err := uc.participantRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return ApproveJoinRequestOutput{}, ErrParticipantNotFound
		}
		return ApproveJoinRequestOutput{}, ErrInvitationCreate
	}

	callerMember, err := uc.chatMemberRepo.FindByRoomAndParticipant(ctx, inv.RoomID, callerParticipant.ID)
	if err != nil {
		if errors.Is(err, chatmember.ErrNotFound) {
			return ApproveJoinRequestOutput{}, ErrNotRoomAdmin
		}
		return ApproveJoinRequestOutput{}, ErrInvitationCreate
	}
	if callerMember.Role != chatmember.Owner && callerMember.Role != chatmember.Admin {
		return ApproveJoinRequestOutput{}, ErrNotRoomAdmin
	}

	if !input.Data.Approve {
		if err := uc.invitationRepo.UpdateStatus(ctx, inv.ID, chatinvitation.Rejected); err != nil {
			return ApproveJoinRequestOutput{}, ErrInvitationCreate
		}
		if err := tx.Commit(); err != nil {
			return ApproveJoinRequestOutput{}, ErrInvitationCreate
		}
		return ApproveJoinRequestOutput{
			InvitationID: int64(inv.ID),
			Status:       string(chatinvitation.Rejected),
		}, nil
	}

	member := &chatmember.ChatMember{
		RoomID:        inv.RoomID,
		ParticipantID: inv.InviteeID,
		Role:          chatmember.Member,
	}
	if err := uc.chatMemberRepo.Add(ctx, member); err != nil {
		return ApproveJoinRequestOutput{}, ErrInvitationCreate
	}

	if err := uc.invitationRepo.UpdateStatus(ctx, inv.ID, chatinvitation.Accepted); err != nil {
		return ApproveJoinRequestOutput{}, ErrInvitationCreate
	}

	if err := tx.Commit(); err != nil {
		return ApproveJoinRequestOutput{}, ErrInvitationCreate
	}

	memberID := int64(member.ID)
	role := string(member.Role)
	return ApproveJoinRequestOutput{
		InvitationID: int64(inv.ID),
		Status:       string(chatinvitation.Accepted),
		MemberID:     &memberID,
		Role:         &role,
		JoinedAt:     &member.JoinedAt,
	}, nil
}

package usecase

import (
	"context"
	"errors"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatinvitation"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
)

type RespondToInvitationInput struct {
	InvitationID int64
	Action       string // "accept" | "reject" | "block"
}

type RespondToInvitationOutput struct {
	InvitationID int64
	Status       string
	MemberID     *int64
	Role         *string
	JoinedAt     *time.Time
}

type RespondToInvitationUseCase struct {
	uow             transaction.UnitOfWork
	participantRepo participant.Repository
	chatMemberRepo  chatmember.Repository
	invitationRepo  chatinvitation.Repository
	friendshipRepo  friendship.Repository
}

func NewRespondToInvitationUseCase(
	uow transaction.UnitOfWork,
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	invitationRepo chatinvitation.Repository,
	friendshipRepo friendship.Repository,
) *RespondToInvitationUseCase {
	return &RespondToInvitationUseCase{
		uow:             uow,
		participantRepo: participantRepo,
		chatMemberRepo:  chatMemberRepo,
		invitationRepo:  invitationRepo,
		friendshipRepo:  friendshipRepo,
	}
}

func (uc *RespondToInvitationUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[RespondToInvitationInput],
) (RespondToInvitationOutput, error) {
	action := input.Data.Action
	if action != "accept" && action != "reject" && action != "block" {
		return RespondToInvitationOutput{}, ErrInvalidInvitationAction
	}

	userID := input.Base.Auth.UserID
	invitationID := chatinvitation.ID(input.Data.InvitationID)

	ctx, tx, err := uc.uow.Begin(ctx)
	if err != nil {
		return RespondToInvitationOutput{}, ErrInvitationCreate
	}
	defer func() { _ = tx.Rollback() }()

	inv, err := uc.invitationRepo.FindByID(ctx, invitationID)
	if err != nil {
		if errors.Is(err, chatinvitation.ErrNotFound) {
			return RespondToInvitationOutput{}, ErrInvitationNotFound
		}
		return RespondToInvitationOutput{}, ErrInvitationCreate
	}

	if inv.Status != chatinvitation.Pending {
		return RespondToInvitationOutput{}, ErrInvitationAlreadyResolved
	}

	if inv.InvitationType != chatinvitation.Invitation {
		return RespondToInvitationOutput{}, ErrInvalidInvitationType
	}

	callerParticipant, err := uc.participantRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return RespondToInvitationOutput{}, ErrParticipantNotFound
		}
		return RespondToInvitationOutput{}, ErrInvitationCreate
	}

	if inv.InviteeID != callerParticipant.ID {
		return RespondToInvitationOutput{}, ErrNotInvitee
	}

	if action == "reject" {
		if err := uc.invitationRepo.UpdateStatus(ctx, inv.ID, chatinvitation.Rejected); err != nil {
			return RespondToInvitationOutput{}, ErrInvitationCreate
		}
		if err := tx.Commit(); err != nil {
			return RespondToInvitationOutput{}, ErrInvitationCreate
		}
		return RespondToInvitationOutput{
			InvitationID: int64(inv.ID),
			Status:       string(chatinvitation.Rejected),
		}, nil
	}

	if action == "block" {
		// Sync block to friendship record so CreateChatRoom also sees it (H-4).
		if uc.friendshipRepo != nil {
			inviterParticipant, pErr := uc.participantRepo.FindByID(ctx, inv.InviterID)
			if pErr == nil && inviterParticipant.UserID != nil {
				fs, fsErr := uc.friendshipRepo.FindBetweenUsers(ctx, userID, *inviterParticipant.UserID)
				if fsErr == nil {
					_ = uc.friendshipRepo.UpdateStatus(ctx, fs.ID, friendship.StatusBlocked)
				} else if errors.Is(fsErr, friendship.ErrFriendshipNotFound) {
					_ = uc.friendshipRepo.CreateBlocked(ctx, userID, *inviterParticipant.UserID)
				}
			}
		}

		if err := uc.invitationRepo.UpdateStatus(ctx, inv.ID, chatinvitation.Blocked); err != nil {
			return RespondToInvitationOutput{}, ErrInvitationCreate
		}
		if err := tx.Commit(); err != nil {
			return RespondToInvitationOutput{}, ErrInvitationCreate
		}
		return RespondToInvitationOutput{
			InvitationID: int64(inv.ID),
			Status:       string(chatinvitation.Blocked),
		}, nil
	}

	// action == "accept" — guard against active membership before upsert (H-7).
	existing, err := uc.chatMemberRepo.FindByRoomAndParticipant(ctx, inv.RoomID, inv.InviteeID)
	if err == nil && !existing.IsDeleted {
		return RespondToInvitationOutput{}, ErrAlreadyMember
	}

	member := &chatmember.ChatMember{
		RoomID:        inv.RoomID,
		ParticipantID: inv.InviteeID,
		Role:          chatmember.Member,
	}
	if err := uc.chatMemberRepo.Add(ctx, member); err != nil {
		return RespondToInvitationOutput{}, ErrInvitationCreate
	}

	if err := uc.invitationRepo.UpdateStatus(ctx, inv.ID, chatinvitation.Accepted); err != nil {
		return RespondToInvitationOutput{}, ErrInvitationCreate
	}

	if err := tx.Commit(); err != nil {
		return RespondToInvitationOutput{}, ErrInvitationCreate
	}

	memberID := int64(member.ID)
	role := string(member.Role)
	return RespondToInvitationOutput{
		InvitationID: int64(inv.ID),
		Status:       string(chatinvitation.Accepted),
		MemberID:     &memberID,
		Role:         &role,
		JoinedAt:     &member.JoinedAt,
	}, nil
}

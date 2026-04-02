package usecase

import (
	"context"
	"errors"
	"fmt"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatinvitation"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type GetMyRoomInvitationInput struct {
	RoomID int64
}

type GetMyRoomInvitationOutput struct {
	Found         bool
	InvitationID  int64
	Role          string  // "inviter" | "invitee"
	InviterName   string  // populated when Role == "invitee"
	InviterAvatar *string // populated when Role == "invitee"
	InviterUserID *int64  // populated when Role == "invitee"; needed for X3DH key exchange
}

type GetMyRoomInvitationUseCase struct {
	participantRepo participant.Repository
	invitationRepo  chatinvitation.Repository
	userRepo        user.Repository
}

func NewGetMyRoomInvitationUseCase(
	participantRepo participant.Repository,
	invitationRepo chatinvitation.Repository,
	userRepo user.Repository,
) *GetMyRoomInvitationUseCase {
	return &GetMyRoomInvitationUseCase{
		participantRepo: participantRepo,
		invitationRepo:  invitationRepo,
		userRepo:        userRepo,
	}
}

func (uc *GetMyRoomInvitationUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[GetMyRoomInvitationInput],
) (GetMyRoomInvitationOutput, error) {
	userID := input.Base.Auth.UserID
	roomID := chatroom.ID(input.Data.RoomID)

	callerParticipant, err := uc.participantRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return GetMyRoomInvitationOutput{}, ErrParticipantNotFound
		}
		return GetMyRoomInvitationOutput{}, fmt.Errorf("%w: find participant for user %d: %w", ErrInvitationQuery, userID, err)
	}

	// Check if caller is the invitee of a pending invitation
	inv, err := uc.invitationRepo.FindByRoomAndInvitee(ctx, roomID, callerParticipant.ID)
	if err == nil {
		out := GetMyRoomInvitationOutput{
			Found:        true,
			InvitationID: int64(inv.ID),
			Role:         "invitee",
		}
		uc.resolveInviterInfo(ctx, inv.InviterID, &out)
		return out, nil
	}
	if !errors.Is(err, chatinvitation.ErrNotFound) {
		return GetMyRoomInvitationOutput{}, fmt.Errorf(
			"%w: find invitee invitation for room %d participant %d: %w",
			ErrInvitationQuery,
			roomID,
			callerParticipant.ID,
			err,
		)
	}

	// Check if caller is the inviter of a pending invitation (to someone else)
	inv, err = uc.invitationRepo.FindPendingByRoomAndInviter(ctx, roomID, callerParticipant.ID)
	if err == nil {
		// Only treat as "inviter waiting" if it's a real invitation (inviter != invitee)
		if inv.InviterID != inv.InviteeID {
			return GetMyRoomInvitationOutput{
				Found:        true,
				InvitationID: int64(inv.ID),
				Role:         "inviter",
			}, nil
		}
	}
	if err != nil && !errors.Is(err, chatinvitation.ErrNotFound) {
		return GetMyRoomInvitationOutput{}, fmt.Errorf(
			"%w: find inviter invitation for room %d participant %d: %w",
			ErrInvitationQuery,
			roomID,
			callerParticipant.ID,
			err,
		)
	}

	return GetMyRoomInvitationOutput{Found: false}, nil
}

func (uc *GetMyRoomInvitationUseCase) resolveInviterInfo(
	ctx context.Context,
	inviterID participant.ID,
	out *GetMyRoomInvitationOutput,
) {
	p, err := uc.participantRepo.FindByID(ctx, inviterID)
	if err != nil || p.UserID == nil {
		return
	}
	userID := int64(*p.UserID)
	out.InviterUserID = &userID

	u, err := uc.userRepo.FindByID(ctx, *p.UserID)
	if err != nil {
		return
	}
	out.InviterName = u.Name
	if u.Avatar != "" {
		avatar := u.Avatar
		out.InviterAvatar = &avatar
	}
}

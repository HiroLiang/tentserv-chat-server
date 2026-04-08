package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	chatPort "github.com/HiroLiang/tentserv-chat-server/internal/application/chat/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatinvitation"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
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
	broadcaster     chatPort.Broadcaster
}

func NewApproveJoinRequestUseCase(
	uow transaction.UnitOfWork,
	chatMemberRepo chatmember.Repository,
	participantRepo participant.Repository,
	invitationRepo chatinvitation.Repository,
	broadcaster chatPort.Broadcaster,
) *ApproveJoinRequestUseCase {
	return &ApproveJoinRequestUseCase{
		uow:             uow,
		chatMemberRepo:  chatMemberRepo,
		participantRepo: participantRepo,
		invitationRepo:  invitationRepo,
		broadcaster:     broadcaster,
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

	if inv.InvitationType != chatinvitation.JoinRequest {
		return ApproveJoinRequestOutput{}, ErrInvalidInvitationType
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

	// Guard against active membership before upsert (H-7).
	existing, err := uc.chatMemberRepo.FindByRoomAndParticipant(ctx, inv.RoomID, inv.InviteeID)
	if err == nil && !existing.IsDeleted {
		return ApproveJoinRequestOutput{}, ErrAlreadyMember
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

	// Notify existing members so they can initiate sender key exchange with the new member.
	go uc.notifyMemberJoined(context.Background(), inv.RoomID, member.ID, inv.InviteeID)

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

func (uc *ApproveJoinRequestUseCase) notifyMemberJoined(
	ctx context.Context,
	roomID chatroom.ID,
	newMemberID chatmember.ID,
	newParticipantID participant.ID,
) {
	if uc.broadcaster == nil {
		return
	}
	members, err := uc.chatMemberRepo.FindByRoom(ctx, roomID)
	if err != nil {
		return
	}
	payload, err := json.Marshal(struct {
		Type    string                `json:"type"`
		Payload wsMemberJoinedPayload `json:"payload"`
	}{
		Type: "chat.member_joined",
		Payload: wsMemberJoinedPayload{
			RoomID:      int64(roomID),
			NewMemberID: int64(newMemberID),
		},
	})
	if err != nil {
		return
	}
	for _, m := range members {
		if m.IsDeleted || m.ParticipantID == newParticipantID {
			continue
		}
		p, err := uc.participantRepo.FindByID(ctx, m.ParticipantID)
		if err != nil || p.UserID == nil {
			continue
		}
		uc.broadcaster.SendToUser(strconv.FormatInt(int64(*p.UserID), 10), payload)
	}
}

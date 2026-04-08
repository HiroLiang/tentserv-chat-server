package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	chatPort "github.com/HiroLiang/tentserv-chat-server/internal/application/chat/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatinvitation"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
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
	broadcaster     chatPort.Broadcaster
}

func NewRespondToInvitationUseCase(
	uow transaction.UnitOfWork,
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	invitationRepo chatinvitation.Repository,
	friendshipRepo friendship.Repository,
	broadcaster chatPort.Broadcaster,
) *RespondToInvitationUseCase {
	return &RespondToInvitationUseCase{
		uow:             uow,
		participantRepo: participantRepo,
		chatMemberRepo:  chatMemberRepo,
		invitationRepo:  invitationRepo,
		friendshipRepo:  friendshipRepo,
		broadcaster:     broadcaster,
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
			if pErr != nil || inviterParticipant.UserID == nil {
				return RespondToInvitationOutput{}, ErrInvitationCreate
			}
			if syncErr := uc.syncBlockedFriendship(ctx, userID, *inviterParticipant.UserID); syncErr != nil {
				return RespondToInvitationOutput{}, ErrInvitationCreate
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

	// Notify existing members so they can initiate sender key exchange with the new member.
	go uc.notifyMemberJoined(context.Background(), inv.RoomID, member.ID, inv.InviteeID)

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

type wsMemberJoinedPayload struct {
	RoomID      int64 `json:"room_id"`
	NewMemberID int64 `json:"new_member_id"`
}

func (uc *RespondToInvitationUseCase) notifyMemberJoined(
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
		Type    string               `json:"type"`
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

func (uc *RespondToInvitationUseCase) syncBlockedFriendship(
	ctx context.Context,
	blockerID shared.UserID,
	targetID shared.UserID,
) error {
	forward, err := uc.friendshipRepo.FindByUserIDAndFriendID(ctx, blockerID, targetID)
	if err == nil {
		if forward.Status != friendship.StatusBlocked {
			if err := uc.friendshipRepo.UpdateStatus(ctx, forward.ID, friendship.StatusBlocked); err != nil {
				return err
			}
		}
	} else if errors.Is(err, friendship.ErrFriendshipNotFound) {
		if err := uc.friendshipRepo.CreateBlocked(ctx, blockerID, targetID); err != nil {
			// Handle concurrent insert: fetch and force blocked.
			if !strings.Contains(strings.ToLower(err.Error()), "duplicate key value") {
				return err
			}
			forward, findErr := uc.friendshipRepo.FindByUserIDAndFriendID(ctx, blockerID, targetID)
			if findErr != nil {
				return findErr
			}
			if forward.Status != friendship.StatusBlocked {
				if err := uc.friendshipRepo.UpdateStatus(ctx, forward.ID, friendship.StatusBlocked); err != nil {
					return err
				}
			}
		}
	} else {
		return err
	}

	reverse, err := uc.friendshipRepo.FindByUserIDAndFriendID(ctx, targetID, blockerID)
	if err == nil {
		if err := uc.friendshipRepo.Delete(ctx, reverse.ID); err != nil {
			return err
		}
		return nil
	}
	if errors.Is(err, friendship.ErrFriendshipNotFound) {
		return nil
	}
	return err
}

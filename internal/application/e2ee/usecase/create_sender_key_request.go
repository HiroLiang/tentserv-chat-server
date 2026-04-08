package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	e2eePort "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyrequest"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type CreateSenderKeyRequestInput struct {
	RoomID           int64
	ProviderMemberID int64
}

type CreateSenderKeyRequestOutput struct{}

type CreateSenderKeyRequestUseCase struct {
	participantRepo      participant.Repository
	chatMemberRepo       chatmember.Repository
	senderKeyRequestRepo senderkeyrequest.Repository
	memberSenderKeyRepo  membersenderkey.Repository
	friendshipRepo       friendship.Repository
	broadcaster          e2eePort.Broadcaster
}

func NewCreateSenderKeyRequestUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	senderKeyRequestRepo senderkeyrequest.Repository,
	memberSenderKeyRepo membersenderkey.Repository,
	friendshipRepo friendship.Repository,
	broadcaster e2eePort.Broadcaster,
) *CreateSenderKeyRequestUseCase {
	return &CreateSenderKeyRequestUseCase{
		participantRepo:      participantRepo,
		chatMemberRepo:       chatMemberRepo,
		senderKeyRequestRepo: senderKeyRequestRepo,
		memberSenderKeyRepo:  memberSenderKeyRepo,
		friendshipRepo:       friendshipRepo,
		broadcaster:          broadcaster,
	}
}

func (u *CreateSenderKeyRequestUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[CreateSenderKeyRequestInput],
) (*CreateSenderKeyRequestOutput, error) {
	callerParticipant, err := u.participantRepo.FindByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, ErrNotRoomMember
	}

	// Verify caller is a member of the room.
	callerMember, err := u.chatMemberRepo.FindByRoomAndParticipant(
		ctx,
		chatroom.ID(input.Data.RoomID),
		callerParticipant.ID,
	)
	if err != nil || callerMember.IsDeleted {
		return nil, ErrNotRoomMember
	}

	// Verify provider is also a member of the room.
	providerMember, err := u.chatMemberRepo.FindByID(ctx, chatmember.ID(input.Data.ProviderMemberID))
	if err != nil || providerMember.IsDeleted {
		return nil, fmt.Errorf("%w: provider not found in room", ErrNotRoomMember)
	}

	// Verify provider belongs to the same room as the requester.
	if providerMember.RoomID != chatroom.ID(input.Data.RoomID) {
		return nil, fmt.Errorf("%w: provider member is not in the requested room", ErrNotRoomMember)
	}

	// If the provider already has an uploaded sender key, the request is unnecessary.
	_, err = u.memberSenderKeyRepo.FindLatest(ctx, providerMember.ID)
	if err == nil {
		// Key already exists — no need to create a request.
		return &CreateSenderKeyRequestOutput{}, nil
	}
	if !errors.Is(err, membersenderkey.ErrNotFound) {
		return nil, fmt.Errorf("create sender key request: check existing key: %w", err)
	}

	// Check for block relationship between the two participants.
	providerParticipant, err := u.participantRepo.FindByID(ctx, providerMember.ParticipantID)
	if err != nil || providerParticipant.UserID == nil {
		return nil, fmt.Errorf("%w: provider participant not found", ErrNotRoomMember)
	}
	if callerParticipant.UserID != nil {
		rows, err := u.friendshipRepo.FindBetweenUsers(ctx, *callerParticipant.UserID, shared.UserID(*providerParticipant.UserID))
		if err == nil {
			for _, f := range rows {
				if f.Status == friendship.StatusBlocked {
					return nil, ErrForbidden
				}
			}
		}
	}

	req := &senderkeyrequest.SenderKeyRequest{
		RequesterMemberID: callerMember.ID,
		ProviderMemberID:  providerMember.ID,
	}
	if err := u.senderKeyRequestRepo.Upsert(ctx, req); err != nil {
		return nil, fmt.Errorf("create sender key request: %w", err)
	}

	// Push e2ee.sender_key_needed to the provider if they are online.
	go u.notifyProvider(context.Background(), providerMember, callerMember, callerParticipant, providerParticipant, input.Data.RoomID)

	return &CreateSenderKeyRequestOutput{}, nil
}

type wsSenderKeyNeededPayload struct {
	RoomID            int64 `json:"room_id"`
	ProviderMemberID  int64 `json:"provider_member_id"`
	RequesterMemberID int64 `json:"requester_member_id"`
	RequesterUserID   int64 `json:"requester_user_id"`
}

func (u *CreateSenderKeyRequestUseCase) notifyProvider(
	_ context.Context,
	providerMember *chatmember.ChatMember,
	requesterMember *chatmember.ChatMember,
	requesterParticipant *participant.Participant,
	providerParticipant *participant.Participant,
	roomID int64,
) {
	if providerParticipant.UserID == nil {
		return
	}

	if requesterParticipant.UserID == nil {
		return
	}

	payload, err := json.Marshal(struct {
		Type    string                   `json:"type"`
		Payload wsSenderKeyNeededPayload `json:"payload"`
	}{
		Type: "e2ee.sender_key_needed",
		Payload: wsSenderKeyNeededPayload{
			RoomID:            roomID,
			ProviderMemberID:  int64(providerMember.ID),
			RequesterMemberID: int64(requesterMember.ID),
			RequesterUserID:   int64(*requesterParticipant.UserID),
		},
	})
	if err != nil {
		return
	}

	providerUserIDStr := strconv.FormatInt(int64(*providerParticipant.UserID), 10)
	u.broadcaster.SendToUser(providerUserIDStr, payload)
}

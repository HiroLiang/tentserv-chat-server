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
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyreceipt"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyrequest"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type CreateSenderKeyRequestInput struct {
	RoomID            int64
	ProviderUserID    int64
	ProviderDeviceID  string
	SenderMemberID    int64
	RequesterDeviceID string
}

type CreateSenderKeyRequestOutput struct{}

type CreateSenderKeyRequestUseCase struct {
	participantRepo      participant.Repository
	chatMemberRepo       chatmember.Repository
	senderKeyRequestRepo senderkeyrequest.Repository
	memberSenderKeyRepo  membersenderkey.Repository
	distributionRepo     senderkeydistribution.Repository
	receiptRepo          senderkeyreceipt.Repository
	selfSyncRepo         selfsenderkeysync.Repository
	friendshipRepo       friendship.Repository
	broadcaster          e2eePort.Broadcaster
}

func NewCreateSenderKeyRequestUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	senderKeyRequestRepo senderkeyrequest.Repository,
	memberSenderKeyRepo membersenderkey.Repository,
	distributionRepo senderkeydistribution.Repository,
	receiptRepo senderkeyreceipt.Repository,
	selfSyncRepo selfsenderkeysync.Repository,
	friendshipRepo friendship.Repository,
	broadcaster e2eePort.Broadcaster,
) *CreateSenderKeyRequestUseCase {
	return &CreateSenderKeyRequestUseCase{
		participantRepo:      participantRepo,
		chatMemberRepo:       chatMemberRepo,
		senderKeyRequestRepo: senderKeyRequestRepo,
		memberSenderKeyRepo:  memberSenderKeyRepo,
		distributionRepo:     distributionRepo,
		receiptRepo:          receiptRepo,
		selfSyncRepo:         selfSyncRepo,
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
	if err := ensureDeviceNotBlockedByActiveSelfSync(ctx, u.selfSyncRepo, callerParticipant.ID, input.Base.Request.DeviceID); err != nil {
		return nil, err
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

	// If a latest distribution is already available for the caller, the request is unnecessary.
	requesterDeviceID, err := resolveRequestedDeviceID(input.Data.RequesterDeviceID, input.Base.Request.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("%w: requester device id", ErrInvalidSignature)
	}

	providerDeviceID, err := shared.ParseDeviceID(input.Data.ProviderDeviceID)
	if err != nil {
		return nil, fmt.Errorf("%w: provider device id", ErrInvalidSignature)
	}

	providerParticipant, err := u.participantRepo.FindByUserID(ctx, shared.UserID(input.Data.ProviderUserID))
	if err != nil || providerParticipant.UserID == nil {
		return nil, fmt.Errorf("%w: provider participant not found", ErrNotRoomMember)
	}

	providerMember, err := u.chatMemberRepo.FindByRoomAndParticipant(
		ctx,
		chatroom.ID(input.Data.RoomID),
		providerParticipant.ID,
	)
	if err != nil || providerMember.IsDeleted {
		return nil, fmt.Errorf("%w: provider not found in room", ErrNotRoomMember)
	}
	if providerMember.ID != chatmember.ID(input.Data.SenderMemberID) {
		return nil, fmt.Errorf("%w: sender member does not match provider user", ErrNotRoomMember)
	}

	latestKey, err := u.memberSenderKeyRepo.FindLatest(ctx, providerMember.ID)
	if err != nil && !errors.Is(err, membersenderkey.ErrNotFound) {
		return nil, fmt.Errorf("create sender key request: check latest sender key: %w", err)
	}
	if err == nil {
		receipt, receiptErr := u.receiptRepo.FindLatest(ctx, providerMember.ID, requesterDeviceID)
		switch {
		case receiptErr == nil && receipt.SenderKeyVersion >= latestKey.SenderKeyVersion:
			if fulfillErr := u.senderKeyRequestRepo.MarkFulfilled(ctx, callerMember.ID, requesterDeviceID, providerMember.ID, providerDeviceID); fulfillErr != nil {
				return nil, fmt.Errorf("create sender key request: mark stale request fulfilled: %w", fulfillErr)
			}
			return &CreateSenderKeyRequestOutput{}, nil
		case receiptErr != nil && !errors.Is(receiptErr, senderkeyreceipt.ErrNotFound):
			return nil, fmt.Errorf("create sender key request: check sender key receipt: %w", receiptErr)
		}
		dist, distErr := u.distributionRepo.FindLatestForReceiver(ctx, providerMember.ID, callerMember.ID, requesterDeviceID)
		if distErr == nil &&
			dist.SenderKeyVersion >= latestKey.SenderKeyVersion &&
			(dist.Status == senderkeydistribution.StatusAvailable || dist.Status == senderkeydistribution.StatusConsumed) {
			if fulfillErr := u.senderKeyRequestRepo.MarkFulfilled(ctx, callerMember.ID, requesterDeviceID, providerMember.ID, providerDeviceID); fulfillErr != nil {
				return nil, fmt.Errorf("create sender key request: mark stale request fulfilled: %w", fulfillErr)
			}
			return &CreateSenderKeyRequestOutput{}, nil
		}
	}

	// Check for block relationship between the two participants.
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
		RequesterDeviceID: requesterDeviceID,
		ProviderMemberID:  providerMember.ID,
		ProviderDeviceID:  providerDeviceID,
	}
	if err := u.senderKeyRequestRepo.Upsert(ctx, req); err != nil {
		return nil, fmt.Errorf("create sender key request: %w", err)
	}

	// Push e2ee.sender_key_needed to the provider if they are online.
	go u.notifyProvider(context.Background(), providerMember, providerDeviceID, callerMember, callerParticipant, providerParticipant, input.Data.RoomID, requesterDeviceID)

	return &CreateSenderKeyRequestOutput{}, nil
}

type wsSenderKeyNeededPayload struct {
	RoomID            int64  `json:"room_id"`
	SenderMemberID    int64  `json:"sender_member_id"`
	ProviderUserID    int64  `json:"provider_user_id"`
	ProviderDeviceID  string `json:"provider_device_id"`
	RequesterMemberID int64  `json:"requester_member_id"`
	RequesterUserID   int64  `json:"requester_user_id"`
	RequesterDeviceID string `json:"requester_device_id,omitempty"`
}

func (u *CreateSenderKeyRequestUseCase) notifyProvider(
	_ context.Context,
	providerMember *chatmember.ChatMember,
	providerDeviceID shared.DeviceID,
	requesterMember *chatmember.ChatMember,
	requesterParticipant *participant.Participant,
	providerParticipant *participant.Participant,
	roomID int64,
	requesterDeviceID shared.DeviceID,
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
			SenderMemberID:    int64(providerMember.ID),
			ProviderUserID:    int64(*providerParticipant.UserID),
			ProviderDeviceID:  providerDeviceID.String(),
			RequesterMemberID: int64(requesterMember.ID),
			RequesterUserID:   int64(*requesterParticipant.UserID),
			RequesterDeviceID: requesterDeviceID.String(),
		},
	})
	if err != nil {
		return
	}

	providerUserIDStr := strconv.FormatInt(int64(*providerParticipant.UserID), 10)
	u.broadcaster.SendToUser(providerUserIDStr, payload)
}

func resolveRequestedDeviceID(explicit string, fallback shared.DeviceID) (shared.DeviceID, error) {
	if explicit != "" {
		return shared.ParseDeviceID(explicit)
	}
	return fallback, nil
}

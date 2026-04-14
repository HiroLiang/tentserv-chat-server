package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	e2eePort "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyrequest"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

// NotifyPendingSenderKeyRequestsUseCase is called when a user connects via WebSocket.
// It finds all pending sender_key_requests where this user is the provider and pushes
// e2ee.sender_key_needed to inform them they need to upload their sender key.
type NotifyPendingSenderKeyRequestsUseCase struct {
	participantRepo      participant.Repository
	chatMemberRepo       chatmember.Repository
	senderKeyRequestRepo senderkeyrequest.Repository
	memberSenderKeyRepo  membersenderkey.Repository
	distributionRepo     senderkeydistribution.Repository
	broadcaster          e2eePort.Broadcaster
}

func NewNotifyPendingSenderKeyRequestsUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	senderKeyRequestRepo senderkeyrequest.Repository,
	memberSenderKeyRepo membersenderkey.Repository,
	distributionRepo senderkeydistribution.Repository,
	broadcaster e2eePort.Broadcaster,
) *NotifyPendingSenderKeyRequestsUseCase {
	return &NotifyPendingSenderKeyRequestsUseCase{
		participantRepo:      participantRepo,
		chatMemberRepo:       chatMemberRepo,
		senderKeyRequestRepo: senderKeyRequestRepo,
		memberSenderKeyRepo:  memberSenderKeyRepo,
		distributionRepo:     distributionRepo,
		broadcaster:          broadcaster,
	}
}

// Execute looks up pending requests for this user (as provider) and pushes notifications.
// userID is a string representation of the user's integer user ID.
func (u *NotifyPendingSenderKeyRequestsUseCase) Execute(ctx context.Context, userIDStr string, providerDeviceID shared.DeviceID) {
	userIDInt, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return
	}
	userID := shared.UserID(userIDInt)

	p, err := u.participantRepo.FindByUserID(ctx, userID)
	if err != nil {
		return
	}

	// Find all chat_members for this participant.
	members, err := u.chatMemberRepo.FindByParticipant(ctx, p.ID)
	if err != nil {
		return
	}

	for _, m := range members {
		if m.IsDeleted {
			continue
		}
		u.notifyForMember(ctx, m, providerDeviceID, userIDStr)
	}
}

func (u *NotifyPendingSenderKeyRequestsUseCase) notifyForMember(
	ctx context.Context,
	providerMember *chatmember.ChatMember,
	providerDeviceID shared.DeviceID,
	providerUserIDStr string,
) {
	requests, err := u.senderKeyRequestRepo.FindPendingByProvider(ctx, providerMember.ID, providerDeviceID)
	if err != nil || len(requests) == 0 {
		return
	}

	for _, req := range requests {
		requesterMember, err := u.chatMemberRepo.FindByID(ctx, req.RequesterMemberID)
		if err != nil || requesterMember.IsDeleted {
			continue
		}
		if u.requestAlreadySatisfied(ctx, providerMember.ID, providerDeviceID, requesterMember.ID, req.RequesterDeviceID) {
			_ = u.senderKeyRequestRepo.MarkFulfilled(ctx, requesterMember.ID, req.RequesterDeviceID, providerMember.ID, providerDeviceID)
			continue
		}
		requesterParticipant, err := u.participantRepo.FindByID(ctx, requesterMember.ParticipantID)
		if err != nil || requesterParticipant.UserID == nil {
			continue
		}

		payload, err := json.Marshal(struct {
			Type    string                   `json:"type"`
			Payload wsSenderKeyNeededPayload `json:"payload"`
		}{
			Type: "e2ee.sender_key_needed",
			Payload: wsSenderKeyNeededPayload{
				RoomID:            int64(requesterMember.RoomID),
				ProviderMemberID:  int64(providerMember.ID),
				ProviderDeviceID:  providerDeviceID.String(),
				RequesterMemberID: int64(requesterMember.ID),
				RequesterUserID:   int64(*requesterParticipant.UserID),
				RequesterDeviceID: req.RequesterDeviceID.String(),
			},
		})
		if err != nil {
			continue
		}

		u.broadcaster.SendToUser(providerUserIDStr, payload)
	}
}

func (u *NotifyPendingSenderKeyRequestsUseCase) requestAlreadySatisfied(
	ctx context.Context,
	providerMemberID chatmember.ID,
	providerDeviceID shared.DeviceID,
	requesterMemberID chatmember.ID,
	requesterDeviceID shared.DeviceID,
) bool {
	latestKey, err := u.memberSenderKeyRepo.FindLatest(ctx, providerMemberID, providerDeviceID)
	if err != nil {
		return false
	}

	dist, err := u.distributionRepo.FindLatest(ctx, providerMemberID, providerDeviceID, requesterMemberID, requesterDeviceID)
	if err != nil {
		if errors.Is(err, senderkeydistribution.ErrNotFound) {
			return false
		}
		return false
	}

	if dist.SenderKeyVersion < latestKey.SenderKeyVersion {
		return false
	}

	return dist.Status == senderkeydistribution.StatusAvailable || dist.Status == senderkeydistribution.StatusConsumed
}

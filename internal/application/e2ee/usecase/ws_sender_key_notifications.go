package usecase

import (
	"context"
	"encoding/json"
	"strconv"

	e2eePort "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
)

type wsSenderKeyDistributionAvailablePayload struct {
	RoomID           int64 `json:"room_id"`
	DistributionID   int64 `json:"distribution_id"`
	SenderMemberID   int64 `json:"sender_member_id"`
	ReceiverMemberID int64 `json:"receiver_member_id"`
	SenderKeyVersion int64 `json:"sender_key_version"`
}

func notifySenderKeyDistributionAvailable(
	ctx context.Context,
	broadcaster e2eePort.Broadcaster,
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	dist *senderkeydistribution.SenderKeyDistribution,
	roomID int64,
) {
	receiverMember, err := chatMemberRepo.FindByID(ctx, dist.ReceiverMemberID)
	if err != nil || receiverMember.IsDeleted {
		return
	}
	receiverParticipant, err := participantRepo.FindByID(ctx, receiverMember.ParticipantID)
	if err != nil || receiverParticipant.UserID == nil {
		return
	}
	userIDStr := strconv.FormatInt(int64(*receiverParticipant.UserID), 10)

	payload, err := json.Marshal(struct {
		Type    string                                  `json:"type"`
		Payload wsSenderKeyDistributionAvailablePayload `json:"payload"`
	}{
		Type: "e2ee.sender_key_distribution_available",
		Payload: wsSenderKeyDistributionAvailablePayload{
			RoomID:           roomID,
			DistributionID:   int64(dist.ID),
			SenderMemberID:   int64(dist.SenderMemberID),
			ReceiverMemberID: int64(dist.ReceiverMemberID),
			SenderKeyVersion: dist.SenderKeyVersion,
		},
	})
	if err == nil {
		broadcaster.SendToUser(userIDStr, payload)
	}

	legacyPayload, err := json.Marshal(struct {
		Type    string `json:"type"`
		Payload struct {
			RoomID           int64 `json:"room_id"`
			ProviderMemberID int64 `json:"provider_member_id"`
		} `json:"payload"`
	}{
		Type: "e2ee.direct_key_ready",
		Payload: struct {
			RoomID           int64 `json:"room_id"`
			ProviderMemberID int64 `json:"provider_member_id"`
		}{
			RoomID:           roomID,
			ProviderMemberID: int64(dist.SenderMemberID),
		},
	})
	if err == nil {
		broadcaster.SendToUser(userIDStr, legacyPayload)
	}
}

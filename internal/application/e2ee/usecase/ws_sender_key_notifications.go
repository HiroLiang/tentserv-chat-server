package usecase

import (
	"context"
	"encoding/json"
	"strconv"

	e2eePort "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"go.uber.org/zap"
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
	if roomID <= 0 {
		logger.Log.Warn("skip sender key distribution available broadcast for invalid room id",
			zap.Int64("room_id", roomID),
			zap.Int64("distribution_id", int64(dist.ID)),
			zap.Int64("sender_member_id", int64(dist.SenderMemberID)),
			zap.Int64("receiver_member_id", int64(dist.ReceiverMemberID)),
		)
		return
	}

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
}

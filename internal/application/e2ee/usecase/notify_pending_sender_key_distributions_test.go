package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotifyPendingSenderKeyDistributions_ReplaysAvailableDistribution(t *testing.T) {
	const (
		receiverUserID   = int64(51)
		roomID           = chatroom.ID(188)
		senderMemberID   = chatmember.ID(341)
		receiverMemberID = chatmember.ID(342)
		distributionID   = senderkeydistribution.ID(901)
		version          = int64(1775880001000)
	)

	receiverUID := shared.UserID(receiverUserID)
	receiverParticipantID := participant.ID(251)

	participantRepo := &senderKeyReqParticipantStub{
		byUserID: map[shared.UserID]*participant.Participant{
			receiverUID: {ID: receiverParticipantID, Type: participant.UserType, UserID: &receiverUID},
		},
		byID: map[participant.ID]*participant.Participant{
			receiverParticipantID: {ID: receiverParticipantID, Type: participant.UserType, UserID: &receiverUID},
		},
	}
	chatMemberRepo := &notifyPendingChatMemberRepoStub{
		byID: map[chatmember.ID]*chatmember.ChatMember{
			receiverMemberID: {ID: receiverMemberID, RoomID: roomID, ParticipantID: receiverParticipantID},
		},
		byParticipant: map[participant.ID][]*chatmember.ChatMember{
			receiverParticipantID: {
				{ID: receiverMemberID, RoomID: roomID, ParticipantID: receiverParticipantID},
			},
		},
	}
	distributionRepo := &notifyPendingDistributionRepoStub{
		latest: map[[2]chatmember.ID]*senderkeydistribution.SenderKeyDistribution{},
		availableByRoomMember: map[[2]int64][]*senderkeydistribution.SenderKeyDistribution{
			{int64(roomID), int64(receiverMemberID)}: {{
				ID:               distributionID,
				SenderMemberID:   senderMemberID,
				ReceiverMemberID: receiverMemberID,
				RoomID:           int64(roomID),
				SenderKeyVersion: version,
			}},
		},
	}
	broadcaster := &senderKeyReqBroadcasterStub{}

	uc := NewNotifyPendingSenderKeyDistributionsUseCase(
		participantRepo,
		chatMemberRepo,
		distributionRepo,
		broadcaster,
	)

	uc.Execute(context.Background(), "51")

	require.Eventually(t, func() bool {
		return broadcaster.callCount() == 1
	}, time.Second, 10*time.Millisecond)
	calls := broadcaster.snapshotCalls()

	var available struct {
		Type    string                                  `json:"type"`
		Payload wsSenderKeyDistributionAvailablePayload `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(calls[0].msg, &available))
	assert.Equal(t, "51", calls[0].userID)
	assert.Equal(t, "e2ee.sender_key_distribution_available", available.Type)
	assert.Equal(t, int64(roomID), available.Payload.RoomID)
	assert.Equal(t, int64(distributionID), available.Payload.DistributionID)
	assert.Equal(t, int64(senderMemberID), available.Payload.SenderMemberID)
	assert.Equal(t, int64(receiverMemberID), available.Payload.ReceiverMemberID)
	assert.Equal(t, version, available.Payload.SenderKeyVersion)
}

func TestNotifyPendingSenderKeyDistributions_SkipsInvalidRoomIDs(t *testing.T) {
	const (
		receiverUserID   = int64(52)
		roomID           = chatroom.ID(0)
		senderMemberID   = chatmember.ID(351)
		receiverMemberID = chatmember.ID(352)
		distributionID   = senderkeydistribution.ID(902)
		version          = int64(1775880002000)
	)

	receiverUID := shared.UserID(receiverUserID)
	receiverParticipantID := participant.ID(252)

	participantRepo := &senderKeyReqParticipantStub{
		byUserID: map[shared.UserID]*participant.Participant{
			receiverUID: {ID: receiverParticipantID, Type: participant.UserType, UserID: &receiverUID},
		},
		byID: map[participant.ID]*participant.Participant{
			receiverParticipantID: {ID: receiverParticipantID, Type: participant.UserType, UserID: &receiverUID},
		},
	}
	chatMemberRepo := &notifyPendingChatMemberRepoStub{
		byID: map[chatmember.ID]*chatmember.ChatMember{
			receiverMemberID: {ID: receiverMemberID, RoomID: roomID, ParticipantID: receiverParticipantID},
		},
		byParticipant: map[participant.ID][]*chatmember.ChatMember{
			receiverParticipantID: {
				{ID: receiverMemberID, RoomID: roomID, ParticipantID: receiverParticipantID},
			},
		},
	}
	distributionRepo := &notifyPendingDistributionRepoStub{
		latest: map[[2]chatmember.ID]*senderkeydistribution.SenderKeyDistribution{},
		availableByRoomMember: map[[2]int64][]*senderkeydistribution.SenderKeyDistribution{
			{int64(roomID), int64(receiverMemberID)}: {{
				ID:               distributionID,
				SenderMemberID:   senderMemberID,
				ReceiverMemberID: receiverMemberID,
				RoomID:           int64(roomID),
				SenderKeyVersion: version,
			}},
		},
	}
	broadcaster := &senderKeyReqBroadcasterStub{}

	uc := NewNotifyPendingSenderKeyDistributionsUseCase(
		participantRepo,
		chatMemberRepo,
		distributionRepo,
		broadcaster,
	)

	uc.Execute(context.Background(), "52")

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, broadcaster.callCount())
}

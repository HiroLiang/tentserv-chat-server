package usecase

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyrequest"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type notifyPendingChatMemberRepoStub struct {
	byID          map[chatmember.ID]*chatmember.ChatMember
	byParticipant map[participant.ID][]*chatmember.ChatMember
}

type notifyPendingSenderKeyRequestRepoStub struct {
	requests []*senderkeyrequest.SenderKeyRequest
}

func (s *notifyPendingChatMemberRepoStub) FindByID(_ context.Context, id chatmember.ID) (*chatmember.ChatMember, error) {
	member, ok := s.byID[id]
	if !ok {
		return nil, chatmember.ErrNotFound
	}
	copied := *member
	return &copied, nil
}

func (s *notifyPendingChatMemberRepoStub) FindByRoomAndParticipant(context.Context, chatroom.ID, participant.ID) (*chatmember.ChatMember, error) {
	return nil, chatmember.ErrNotFound
}

func (s *notifyPendingChatMemberRepoStub) FindByRoom(context.Context, chatroom.ID) ([]*chatmember.ChatMember, error) {
	return nil, nil
}

func (s *notifyPendingChatMemberRepoStub) FindByParticipant(_ context.Context, participantID participant.ID) ([]*chatmember.ChatMember, error) {
	members := s.byParticipant[participantID]
	out := make([]*chatmember.ChatMember, 0, len(members))
	for _, member := range members {
		copied := *member
		out = append(out, &copied)
	}
	return out, nil
}

func (s *notifyPendingChatMemberRepoStub) Add(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (s *notifyPendingChatMemberRepoStub) Update(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (s *notifyPendingChatMemberRepoStub) SoftDelete(context.Context, chatmember.ID) error {
	return nil
}

func (s *notifyPendingChatMemberRepoStub) Remove(context.Context, chatroom.ID, participant.ID) error {
	return nil
}

func (s *notifyPendingSenderKeyRequestRepoStub) Upsert(context.Context, *senderkeyrequest.SenderKeyRequest) error {
	return nil
}

func (s *notifyPendingSenderKeyRequestRepoStub) FindPendingByProvider(_ context.Context, providerID chatmember.ID) ([]*senderkeyrequest.SenderKeyRequest, error) {
	out := make([]*senderkeyrequest.SenderKeyRequest, 0, len(s.requests))
	for _, req := range s.requests {
		if req.ProviderMemberID != providerID || req.FulfilledAt != nil {
			continue
		}
		copied := *req
		out = append(out, &copied)
	}
	return out, nil
}

func (s *notifyPendingSenderKeyRequestRepoStub) MarkFulfilled(context.Context, chatmember.ID, chatmember.ID) error {
	return nil
}

func TestNotifyPendingSenderKeyRequests_NotifyForMemberReplaysPendingRequest(t *testing.T) {
	const (
		providerUserID  = int64(41)
		requesterUserID = int64(42)
		roomID          = chatroom.ID(88)
	)

	providerUID := shared.UserID(providerUserID)
	requesterUID := shared.UserID(requesterUserID)
	providerParticipantID := participant.ID(141)
	requesterParticipantID := participant.ID(142)
	providerMemberID := chatmember.ID(241)
	requesterMemberID := chatmember.ID(242)

	participantRepo := &senderKeyReqParticipantStub{
		byUserID: map[shared.UserID]*participant.Participant{
			providerUID:  {ID: providerParticipantID, Type: participant.UserType, UserID: &providerUID},
			requesterUID: {ID: requesterParticipantID, Type: participant.UserType, UserID: &requesterUID},
		},
		byID: map[participant.ID]*participant.Participant{
			providerParticipantID:  {ID: providerParticipantID, Type: participant.UserType, UserID: &providerUID},
			requesterParticipantID: {ID: requesterParticipantID, Type: participant.UserType, UserID: &requesterUID},
		},
	}
	chatMemberRepo := &notifyPendingChatMemberRepoStub{
		byID: map[chatmember.ID]*chatmember.ChatMember{
			providerMemberID:  {ID: providerMemberID, RoomID: roomID, ParticipantID: providerParticipantID},
			requesterMemberID: {ID: requesterMemberID, RoomID: roomID, ParticipantID: requesterParticipantID},
		},
		byParticipant: map[participant.ID][]*chatmember.ChatMember{
			providerParticipantID: {
				{ID: providerMemberID, RoomID: roomID, ParticipantID: providerParticipantID},
			},
		},
	}
	requestRepo := &notifyPendingSenderKeyRequestRepoStub{
		requests: []*senderkeyrequest.SenderKeyRequest{{
			RequesterMemberID: requesterMemberID,
			ProviderMemberID:  providerMemberID,
		}},
	}
	broadcaster := &senderKeyReqBroadcasterStub{}

	uc := NewNotifyPendingSenderKeyRequestsUseCase(
		participantRepo,
		chatMemberRepo,
		requestRepo,
		broadcaster,
	)

	uc.notifyForMember(context.Background(), &chatmember.ChatMember{
		ID:            providerMemberID,
		RoomID:        roomID,
		ParticipantID: providerParticipantID,
	}, "41")

	require.Equal(t, 1, broadcaster.callCount())
	call := broadcaster.lastCall()
	assert.Equal(t, "41", call.userID)

	var envelope struct {
		Type    string `json:"type"`
		Payload struct {
			RoomID            int64 `json:"room_id"`
			ProviderMemberID  int64 `json:"provider_member_id"`
			RequesterMemberID int64 `json:"requester_member_id"`
			RequesterUserID   int64 `json:"requester_user_id"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(call.msg, &envelope))
	assert.Equal(t, "e2ee.sender_key_needed", envelope.Type)
	assert.Equal(t, int64(roomID), envelope.Payload.RoomID)
	assert.Equal(t, int64(providerMemberID), envelope.Payload.ProviderMemberID)
	assert.Equal(t, int64(requesterMemberID), envelope.Payload.RequesterMemberID)
	assert.Equal(t, requesterUserID, envelope.Payload.RequesterUserID)
}

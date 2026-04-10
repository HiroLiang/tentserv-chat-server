package usecase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyrequest"
	sharedDomain "github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type uploadSenderKeyRequestRepoStub struct {
	markFulfilledFor [][2]chatmember.ID
}

func (s *uploadSenderKeyRequestRepoStub) Upsert(context.Context, *senderkeyrequest.SenderKeyRequest) error {
	return nil
}

func (s *uploadSenderKeyRequestRepoStub) FindPendingByProvider(context.Context, chatmember.ID) ([]*senderkeyrequest.SenderKeyRequest, error) {
	return nil, nil
}

func (s *uploadSenderKeyRequestRepoStub) MarkFulfilled(_ context.Context, requesterMemberID, providerMemberID chatmember.ID) error {
	s.markFulfilledFor = append(s.markFulfilledFor, [2]chatmember.ID{requesterMemberID, providerMemberID})
	return nil
}

type uploadSenderKeyMemberSenderKeyRepoStub struct {
	upsertErr error
	upserted  []*membersenderkey.MemberSenderKey
}

func (s *uploadSenderKeyMemberSenderKeyRepoStub) FindLatest(context.Context, chatmember.ID) (*membersenderkey.MemberSenderKey, error) {
	return nil, membersenderkey.ErrNotFound
}

func (s *uploadSenderKeyMemberSenderKeyRepoStub) FindAllByMembers(context.Context, []chatmember.ID) ([]*membersenderkey.MemberSenderKey, error) {
	return nil, nil
}

func (s *uploadSenderKeyMemberSenderKeyRepoStub) Add(_ context.Context, sk *membersenderkey.MemberSenderKey) error {
	return s.UpsertLatest(context.Background(), sk)
}

func (s *uploadSenderKeyMemberSenderKeyRepoStub) UpsertLatest(_ context.Context, sk *membersenderkey.MemberSenderKey) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	copied := *sk
	s.upserted = append(s.upserted, &copied)
	return nil
}

type uploadSenderKeyDistributionRepoStub struct {
	upsertErr error
	upserted  []*senderkeydistribution.SenderKeyDistribution
}

func (s *uploadSenderKeyDistributionRepoStub) UpsertBatch(context.Context, []*senderkeydistribution.SenderKeyDistribution) error {
	return nil
}

func (s *uploadSenderKeyDistributionRepoStub) FindPendingReceivers(context.Context, chatmember.ID, int64) ([]chatmember.ID, error) {
	return nil, nil
}

func (s *uploadSenderKeyDistributionRepoStub) UpsertAvailable(_ context.Context, dist *senderkeydistribution.SenderKeyDistribution) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	copied := *dist
	copied.ID = senderkeydistribution.ID(len(s.upserted) + 1)
	copied.DistributionMessage = append([]byte(nil), dist.DistributionMessage...)
	s.upserted = append(s.upserted, &copied)
	dist.ID = copied.ID
	return nil
}

func (s *uploadSenderKeyDistributionRepoStub) FindLatest(context.Context, chatmember.ID, chatmember.ID) (*senderkeydistribution.SenderKeyDistribution, error) {
	return nil, senderkeydistribution.ErrNotFound
}

func (s *uploadSenderKeyDistributionRepoStub) FindAvailableByRoomAndReceiver(context.Context, chatroom.ID, chatmember.ID) ([]*senderkeydistribution.SenderKeyDistribution, error) {
	return nil, nil
}

func (s *uploadSenderKeyDistributionRepoStub) FindByID(context.Context, senderkeydistribution.ID) (*senderkeydistribution.SenderKeyDistribution, error) {
	return nil, senderkeydistribution.ErrNotFound
}

func (s *uploadSenderKeyDistributionRepoStub) MarkConsumed(context.Context, senderkeydistribution.ID) error {
	return nil
}

func (s *uploadSenderKeyDistributionRepoStub) MarkFailed(context.Context, senderkeydistribution.ID) error {
	return nil
}

func makeUploadSenderKeyInput(callerUserID int64, roomID int64, receiverMemberID int64, senderKeyVersion int64, distributionMessage string) appShared.UseCaseInput[UploadSenderKeyInput] {
	uid := sharedDomain.UserID(callerUserID)
	auth := appShared.AuthContext{UserID: uid}
	return appShared.UseCaseInput[UploadSenderKeyInput]{
		Base: appShared.BaseContext{Auth: &auth},
		Data: UploadSenderKeyInput{
			RoomID:              roomID,
			ReceiverMemberID:    receiverMemberID,
			SenderKeyVersion:    senderKeyVersion,
			DistributionMessage: distributionMessage,
		},
	}
}

func encodeB64(bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}

func TestUploadSenderKey_SuccessNotifiesAndMarksReceiverFulfilled(t *testing.T) {
	const (
		providerUID  = int64(21)
		requesterUID = int64(22)
		roomID       = chatroom.ID(200)
	)

	participantRepo := makeParticipantStub(providerUID, requesterUID)
	chatMemberRepo, providerMemberID, requesterMemberID := makeChatMemberStub(roomID, providerUID, requesterUID, roomID)
	memberSenderKeyRepo := &uploadSenderKeyMemberSenderKeyRepoStub{}
	distributionRepo := &uploadSenderKeyDistributionRepoStub{}
	requestRepo := &uploadSenderKeyRequestRepoStub{}
	broadcaster := &senderKeyReqBroadcasterStub{}

	uc := NewUploadSenderKeyUseCase(
		participantRepo,
		chatMemberRepo,
		memberSenderKeyRepo,
		distributionRepo,
		requestRepo,
		&senderKeyReqFriendshipStub{},
		broadcaster,
	)

	distBytes := []byte(`{"ciphertext":"dist"}`)
	const senderKeyVersion = int64(1700000000000)

	_, err := uc.Execute(context.Background(), makeUploadSenderKeyInput(
		providerUID,
		int64(roomID),
		int64(requesterMemberID),
		senderKeyVersion,
		encodeB64(distBytes),
	))

	require.NoError(t, err)
	require.Len(t, memberSenderKeyRepo.upserted, 1)
	assert.Equal(t, providerMemberID, memberSenderKeyRepo.upserted[0].ChatMemberID)
	assert.Equal(t, senderKeyVersion, memberSenderKeyRepo.upserted[0].SenderKeyVersion)
	assert.Equal(t, membersenderkey.ChainID(senderKeyVersion), memberSenderKeyRepo.upserted[0].ChainID)

	require.Len(t, distributionRepo.upserted, 1)
	assert.Equal(t, providerMemberID, distributionRepo.upserted[0].SenderMemberID)
	assert.Equal(t, requesterMemberID, distributionRepo.upserted[0].ReceiverMemberID)
	assert.Equal(t, int64(roomID), distributionRepo.upserted[0].RoomID)
	assert.Equal(t, senderKeyVersion, distributionRepo.upserted[0].SenderKeyVersion)
	assert.Equal(t, senderKeyVersion, distributionRepo.upserted[0].ChainID)
	assert.Equal(t, distBytes, distributionRepo.upserted[0].DistributionMessage)
	assert.Equal(t, senderkeydistribution.StatusAvailable, distributionRepo.upserted[0].Status)

	require.Eventually(t, func() bool {
		return broadcaster.callCount() == 2 && len(requestRepo.markFulfilledFor) == 1
	}, time.Second, 10*time.Millisecond)

	assert.Equal(t, [2]chatmember.ID{requesterMemberID, providerMemberID}, requestRepo.markFulfilledFor[0])

	broadcaster.mu.Lock()
	calls := append([]senderKeyReqBroadcastCall(nil), broadcaster.calls...)
	broadcaster.mu.Unlock()
	require.Len(t, calls, 2)
	assert.Equal(t, fmt.Sprint(requesterUID), calls[0].userID)
	assert.Equal(t, fmt.Sprint(requesterUID), calls[1].userID)

	var availableMsg struct {
		Type    string `json:"type"`
		Payload struct {
			RoomID           int64 `json:"room_id"`
			DistributionID   int64 `json:"distribution_id"`
			SenderMemberID   int64 `json:"sender_member_id"`
			ReceiverMemberID int64 `json:"receiver_member_id"`
			SenderKeyVersion int64 `json:"sender_key_version"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(calls[0].msg, &availableMsg))
	assert.Equal(t, "e2ee.sender_key_distribution_available", availableMsg.Type)
	assert.Equal(t, int64(roomID), availableMsg.Payload.RoomID)
	assert.Equal(t, int64(distributionRepo.upserted[0].ID), availableMsg.Payload.DistributionID)
	assert.Equal(t, int64(providerMemberID), availableMsg.Payload.SenderMemberID)
	assert.Equal(t, int64(requesterMemberID), availableMsg.Payload.ReceiverMemberID)
	assert.Equal(t, senderKeyVersion, availableMsg.Payload.SenderKeyVersion)

	var legacyMsg struct {
		Type    string `json:"type"`
		Payload struct {
			RoomID           int64 `json:"room_id"`
			ProviderMemberID int64 `json:"provider_member_id"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(calls[1].msg, &legacyMsg))
	assert.Equal(t, "e2ee.direct_key_ready", legacyMsg.Type)
	assert.Equal(t, int64(roomID), legacyMsg.Payload.RoomID)
	assert.Equal(t, int64(providerMemberID), legacyMsg.Payload.ProviderMemberID)
}

func TestUploadSenderKey_InvalidDistributionMessage(t *testing.T) {
	const (
		providerUID = int64(31)
		roomID      = chatroom.ID(300)
	)

	participantRepo := makeParticipantStub(providerUID, 32)
	chatMemberRepo, providerMemberID, _ := makeChatMemberStub(roomID, providerUID, 32, roomID)
	_ = providerMemberID

	uc := NewUploadSenderKeyUseCase(
		participantRepo,
		chatMemberRepo,
		&uploadSenderKeyMemberSenderKeyRepoStub{},
		&uploadSenderKeyDistributionRepoStub{},
		&uploadSenderKeyRequestRepoStub{},
		&senderKeyReqFriendshipStub{},
		&senderKeyReqBroadcasterStub{},
	)

	_, err := uc.Execute(context.Background(), makeUploadSenderKeyInput(
		providerUID,
		int64(roomID),
		132,
		1700000000001,
		"invalid-dist",
	))

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidSignature), "expected ErrInvalidSignature, got %v", err)
}

func TestUploadSenderKey_CallerNotInRoom(t *testing.T) {
	const (
		providerUID = int64(51)
		roomID      = chatroom.ID(500)
	)

	participantRepo := makeParticipantStub(providerUID, 52)
	chatMemberRepo := &senderKeyReqChatMemberStub{
		byID:                 map[chatmember.ID]*chatmember.ChatMember{},
		byRoomAndParticipant: map[chatroom.ID]map[participant.ID]*chatmember.ChatMember{},
	}

	uc := NewUploadSenderKeyUseCase(
		participantRepo,
		chatMemberRepo,
		&uploadSenderKeyMemberSenderKeyRepoStub{},
		&uploadSenderKeyDistributionRepoStub{},
		&uploadSenderKeyRequestRepoStub{},
		&senderKeyReqFriendshipStub{},
		&senderKeyReqBroadcasterStub{},
	)

	_, err := uc.Execute(context.Background(), makeUploadSenderKeyInput(
		providerUID,
		int64(roomID),
		151,
		1700000000002,
		encodeB64([]byte("dist")),
	))

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotRoomMember), "expected ErrNotRoomMember, got %v", err)
}

func TestUploadSenderKey_BlockedRelationship(t *testing.T) {
	const (
		providerUID  = int64(56)
		requesterUID = int64(57)
		roomID       = chatroom.ID(560)
	)

	providerUserID := sharedDomain.UserID(providerUID)
	requesterUserID := sharedDomain.UserID(requesterUID)

	participantRepo := makeParticipantStub(providerUID, requesterUID)
	chatMemberRepo, _, requesterMemberID := makeChatMemberStub(roomID, providerUID, requesterUID, roomID)

	uc := NewUploadSenderKeyUseCase(
		participantRepo,
		chatMemberRepo,
		&uploadSenderKeyMemberSenderKeyRepoStub{},
		&uploadSenderKeyDistributionRepoStub{},
		&uploadSenderKeyRequestRepoStub{},
		&senderKeyReqFriendshipStub{
			rows: []*friendship.Friendship{{
				Status:   friendship.StatusBlocked,
				UserID:   providerUserID,
				FriendID: requesterUserID,
			}},
		},
		&senderKeyReqBroadcasterStub{},
	)

	_, err := uc.Execute(context.Background(), makeUploadSenderKeyInput(
		providerUID,
		int64(roomID),
		int64(requesterMemberID),
		1700000000003,
		encodeB64([]byte("dist")),
	))

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrForbidden), "expected ErrForbidden, got %v", err)
}

func TestUploadSenderKey_UpsertMetadataError(t *testing.T) {
	const (
		providerUID = int64(61)
		roomID      = chatroom.ID(600)
	)

	participantRepo := makeParticipantStub(providerUID, 62)
	chatMemberRepo, _, _ := makeChatMemberStub(roomID, providerUID, 62, roomID)
	memberSenderKeyRepo := &uploadSenderKeyMemberSenderKeyRepoStub{upsertErr: errors.New("add failed")}

	uc := NewUploadSenderKeyUseCase(
		participantRepo,
		chatMemberRepo,
		memberSenderKeyRepo,
		&uploadSenderKeyDistributionRepoStub{},
		&uploadSenderKeyRequestRepoStub{},
		&senderKeyReqFriendshipStub{},
		&senderKeyReqBroadcasterStub{},
	)

	_, err := uc.Execute(context.Background(), makeUploadSenderKeyInput(
		providerUID,
		int64(roomID),
		162,
		1700000000004,
		encodeB64([]byte("dist")),
	))

	require.Error(t, err)
	assert.EqualError(t, err, "upsert sender key metadata: add failed")
}

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
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyrequest"
	sharedDomain "github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type uploadSenderKeyRequestRepoStub struct {
	pending          map[chatmember.ID][]*senderkeyrequest.SenderKeyRequest
	findPendingErr   error
	markFulfilledFor [][2]chatmember.ID
}

func (s *uploadSenderKeyRequestRepoStub) Upsert(context.Context, *senderkeyrequest.SenderKeyRequest) error {
	return nil
}

func (s *uploadSenderKeyRequestRepoStub) FindPendingByProvider(_ context.Context, providerMemberID chatmember.ID) ([]*senderkeyrequest.SenderKeyRequest, error) {
	if s.findPendingErr != nil {
		return nil, s.findPendingErr
	}
	requests := s.pending[providerMemberID]
	out := make([]*senderkeyrequest.SenderKeyRequest, len(requests))
	for i, req := range requests {
		copied := *req
		out[i] = &copied
	}
	return out, nil
}

func (s *uploadSenderKeyRequestRepoStub) MarkFulfilled(_ context.Context, requesterMemberID, providerMemberID chatmember.ID) error {
	s.markFulfilledFor = append(s.markFulfilledFor, [2]chatmember.ID{requesterMemberID, providerMemberID})
	return nil
}

type uploadSenderKeyMemberSenderKeyRepoStub struct {
	addErr error
	added  []*membersenderkey.MemberSenderKey
}

func (s *uploadSenderKeyMemberSenderKeyRepoStub) FindLatest(context.Context, chatmember.ID) (*membersenderkey.MemberSenderKey, error) {
	return nil, membersenderkey.ErrNotFound
}

func (s *uploadSenderKeyMemberSenderKeyRepoStub) FindAllByMembers(context.Context, []chatmember.ID) ([]*membersenderkey.MemberSenderKey, error) {
	return nil, nil
}

func (s *uploadSenderKeyMemberSenderKeyRepoStub) Add(_ context.Context, sk *membersenderkey.MemberSenderKey) error {
	if s.addErr != nil {
		return s.addErr
	}
	copied := *sk
	copied.DistributionMessage = append([]byte(nil), sk.DistributionMessage...)
	s.added = append(s.added, &copied)
	return nil
}

func makeUploadSenderKeyInput(callerUserID int64, roomID int64, senderKeyPublic string, distributionMessage string) appShared.UseCaseInput[UploadSenderKeyInput] {
	uid := sharedDomain.UserID(callerUserID)
	auth := appShared.AuthContext{UserID: uid}
	return appShared.UseCaseInput[UploadSenderKeyInput]{
		Base: appShared.BaseContext{Auth: &auth},
		Data: UploadSenderKeyInput{
			RoomID:              roomID,
			SenderKeyPublic:     senderKeyPublic,
			DistributionMessage: distributionMessage,
		},
	}
}

func encodeB64(bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}

func TestUploadSenderKey_SuccessNotifiesAndFulfillsRequesters(t *testing.T) {
	const (
		providerUID  = int64(21)
		requesterUID = int64(22)
		roomID       = chatroom.ID(200)
	)

	participantRepo := makeParticipantStub(providerUID, requesterUID)
	chatMemberRepo, providerMemberID, requesterMemberID := makeChatMemberStub(roomID, providerUID, requesterUID, roomID)
	memberSenderKeyRepo := &uploadSenderKeyMemberSenderKeyRepoStub{}
	requestRepo := &uploadSenderKeyRequestRepoStub{
		pending: map[chatmember.ID][]*senderkeyrequest.SenderKeyRequest{
			providerMemberID: {
				{
					RequesterMemberID: requesterMemberID,
					ProviderMemberID:  providerMemberID,
				},
			},
		},
	}
	broadcaster := &senderKeyReqBroadcasterStub{}

	uc := NewUploadSenderKeyUseCase(
		participantRepo,
		chatMemberRepo,
		memberSenderKeyRepo,
		requestRepo,
		broadcaster,
	)

	pubBytes := senderKeyBytes(9, 32)
	distBytes := []byte(`{"ciphertext":"dist"}`)

	_, err := uc.Execute(context.Background(), makeUploadSenderKeyInput(
		providerUID,
		int64(roomID),
		encodeB64(pubBytes),
		encodeB64(distBytes),
	))

	require.NoError(t, err)
	require.Len(t, memberSenderKeyRepo.added, 1)
	assert.Equal(t, providerMemberID, memberSenderKeyRepo.added[0].ChatMemberID)
	assert.Equal(t, distBytes, memberSenderKeyRepo.added[0].DistributionMessage)
	assert.Equal(t, pubBytes, memberSenderKeyRepo.added[0].SenderKeyPublic[:])

	require.Eventually(t, func() bool {
		return broadcaster.callCount() == 1 && len(requestRepo.markFulfilledFor) == 1
	}, time.Second, 10*time.Millisecond)

	call := broadcaster.lastCall()
	assert.Equal(t, fmt.Sprint(requesterUID), call.userID)

	var msg struct {
		Type    string `json:"type"`
		Payload struct {
			RoomID           int64 `json:"room_id"`
			ProviderMemberID int64 `json:"provider_member_id"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(call.msg, &msg))
	assert.Equal(t, "e2ee.direct_key_ready", msg.Type)
	assert.Equal(t, int64(roomID), msg.Payload.RoomID)
	assert.Equal(t, int64(providerMemberID), msg.Payload.ProviderMemberID)
	assert.Equal(t, [2]chatmember.ID{requesterMemberID, providerMemberID}, requestRepo.markFulfilledFor[0])
}

func TestUploadSenderKey_InvalidSenderKey(t *testing.T) {
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
		&uploadSenderKeyRequestRepoStub{},
		&senderKeyReqBroadcasterStub{},
	)

	_, err := uc.Execute(context.Background(), makeUploadSenderKeyInput(
		providerUID,
		int64(roomID),
		"invalid-base64",
		encodeB64([]byte("dist")),
	))

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidSignature), "expected ErrInvalidSignature, got %v", err)
}

func TestUploadSenderKey_InvalidDistributionMessage(t *testing.T) {
	const (
		providerUID = int64(41)
		roomID      = chatroom.ID(400)
	)

	participantRepo := makeParticipantStub(providerUID, 42)
	chatMemberRepo, _, _ := makeChatMemberStub(roomID, providerUID, 42, roomID)

	uc := NewUploadSenderKeyUseCase(
		participantRepo,
		chatMemberRepo,
		&uploadSenderKeyMemberSenderKeyRepoStub{},
		&uploadSenderKeyRequestRepoStub{},
		&senderKeyReqBroadcasterStub{},
	)

	_, err := uc.Execute(context.Background(), makeUploadSenderKeyInput(
		providerUID,
		int64(roomID),
		encodeB64(senderKeyBytes(7, 32)),
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
		&uploadSenderKeyRequestRepoStub{},
		&senderKeyReqBroadcasterStub{},
	)

	_, err := uc.Execute(context.Background(), makeUploadSenderKeyInput(
		providerUID,
		int64(roomID),
		encodeB64(senderKeyBytes(8, 32)),
		encodeB64([]byte("dist")),
	))

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotRoomMember), "expected ErrNotRoomMember, got %v", err)
}

func TestUploadSenderKey_AddError(t *testing.T) {
	const (
		providerUID = int64(61)
		roomID      = chatroom.ID(600)
	)

	participantRepo := makeParticipantStub(providerUID, 62)
	chatMemberRepo, _, _ := makeChatMemberStub(roomID, providerUID, 62, roomID)
	memberSenderKeyRepo := &uploadSenderKeyMemberSenderKeyRepoStub{addErr: errors.New("add failed")}

	uc := NewUploadSenderKeyUseCase(
		participantRepo,
		chatMemberRepo,
		memberSenderKeyRepo,
		&uploadSenderKeyRequestRepoStub{},
		&senderKeyReqBroadcasterStub{},
	)

	_, err := uc.Execute(context.Background(), makeUploadSenderKeyInput(
		providerUID,
		int64(roomID),
		encodeB64(senderKeyBytes(6, 32)),
		encodeB64([]byte("dist")),
	))

	require.Error(t, err)
	assert.EqualError(t, err, "add sender key: add failed")
}

func senderKeyBytes(value byte, length int) []byte {
	out := make([]byte, length)
	for i := range out {
		out[i] = value
	}
	return out
}

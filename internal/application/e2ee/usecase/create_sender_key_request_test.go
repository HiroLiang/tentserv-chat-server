package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	e2eePort "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/port"
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

type senderKeyReqParticipantStub struct {
	byUserID map[sharedDomain.UserID]*participant.Participant
	byID     map[participant.ID]*participant.Participant
}

func (s *senderKeyReqParticipantStub) FindByID(_ context.Context, id participant.ID) (*participant.Participant, error) {
	p, ok := s.byID[id]
	if !ok {
		return nil, participant.ErrNotFound
	}
	return p, nil
}

func (s *senderKeyReqParticipantStub) FindByUserID(_ context.Context, userID sharedDomain.UserID) (*participant.Participant, error) {
	p, ok := s.byUserID[userID]
	if !ok {
		return nil, participant.ErrNotFound
	}
	return p, nil
}

func (s *senderKeyReqParticipantStub) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *senderKeyReqParticipantStub) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *senderKeyReqParticipantStub) Create(context.Context, *participant.Participant) error {
	return nil
}

type senderKeyReqChatMemberStub struct {
	byID                 map[chatmember.ID]*chatmember.ChatMember
	byRoomAndParticipant map[chatroom.ID]map[participant.ID]*chatmember.ChatMember
}

func (s *senderKeyReqChatMemberStub) FindByID(_ context.Context, id chatmember.ID) (*chatmember.ChatMember, error) {
	m, ok := s.byID[id]
	if !ok {
		return nil, chatmember.ErrNotFound
	}
	return m, nil
}

func (s *senderKeyReqChatMemberStub) FindByRoomAndParticipant(_ context.Context, roomID chatroom.ID, pID participant.ID) (*chatmember.ChatMember, error) {
	if roomMap, ok := s.byRoomAndParticipant[roomID]; ok {
		if m, ok := roomMap[pID]; ok {
			return m, nil
		}
	}
	return nil, chatmember.ErrNotFound
}

func (s *senderKeyReqChatMemberStub) FindByRoom(context.Context, chatroom.ID) ([]*chatmember.ChatMember, error) {
	return nil, nil
}

func (s *senderKeyReqChatMemberStub) FindByParticipant(context.Context, participant.ID) ([]*chatmember.ChatMember, error) {
	return nil, nil
}

func (s *senderKeyReqChatMemberStub) Add(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (s *senderKeyReqChatMemberStub) Update(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (s *senderKeyReqChatMemberStub) SoftDelete(context.Context, chatmember.ID) error {
	return nil
}

func (s *senderKeyReqChatMemberStub) Remove(context.Context, chatroom.ID, participant.ID) error {
	return nil
}

type senderKeyReqSKRRepoStub struct {
	records     map[string]*senderkeyrequest.SenderKeyRequest
	upsertCount int
}

func (s *senderKeyReqSKRRepoStub) Upsert(_ context.Context, req *senderkeyrequest.SenderKeyRequest) error {
	if s.records == nil {
		s.records = map[string]*senderkeyrequest.SenderKeyRequest{}
	}
	copied := *req
	s.records[s.key(req.RequesterMemberID, req.ProviderMemberID)] = &copied
	s.upsertCount++
	return nil
}

func (s *senderKeyReqSKRRepoStub) FindPendingByProvider(context.Context, chatmember.ID) ([]*senderkeyrequest.SenderKeyRequest, error) {
	return nil, nil
}

func (s *senderKeyReqSKRRepoStub) MarkFulfilled(context.Context, chatmember.ID, chatmember.ID) error {
	return nil
}

func (s *senderKeyReqSKRRepoStub) key(requesterMemberID, providerMemberID chatmember.ID) string {
	return fmt.Sprintf("%d:%d", requesterMemberID, providerMemberID)
}

func (s *senderKeyReqSKRRepoStub) storedRequest(requesterMemberID, providerMemberID chatmember.ID) *senderkeyrequest.SenderKeyRequest {
	if s.records == nil {
		return nil
	}
	return s.records[s.key(requesterMemberID, providerMemberID)]
}

func (s *senderKeyReqSKRRepoStub) storedCount() int {
	return len(s.records)
}

type senderKeyReqMSKRepoStub struct {
	findLatestErr error
}

func (s *senderKeyReqMSKRepoStub) FindLatest(context.Context, chatmember.ID) (*membersenderkey.MemberSenderKey, error) {
	if s.findLatestErr != nil {
		return nil, s.findLatestErr
	}
	return &membersenderkey.MemberSenderKey{}, nil
}

func (s *senderKeyReqMSKRepoStub) FindAllByMembers(context.Context, []chatmember.ID) ([]*membersenderkey.MemberSenderKey, error) {
	return nil, nil
}

func (s *senderKeyReqMSKRepoStub) Add(context.Context, *membersenderkey.MemberSenderKey) error {
	return nil
}

func (s *senderKeyReqMSKRepoStub) UpsertLatest(context.Context, *membersenderkey.MemberSenderKey) error {
	return nil
}

type senderKeyReqDistributionRepoStub struct {
	latest map[string]*senderkeydistribution.SenderKeyDistribution
}

func (s *senderKeyReqDistributionRepoStub) UpsertBatch(context.Context, []*senderkeydistribution.SenderKeyDistribution) error {
	return nil
}

func (s *senderKeyReqDistributionRepoStub) FindPendingReceivers(context.Context, chatmember.ID, int64) ([]chatmember.ID, error) {
	return nil, nil
}

func (s *senderKeyReqDistributionRepoStub) UpsertAvailable(context.Context, *senderkeydistribution.SenderKeyDistribution) error {
	return nil
}

func (s *senderKeyReqDistributionRepoStub) FindLatest(_ context.Context, senderMemberID, receiverMemberID chatmember.ID) (*senderkeydistribution.SenderKeyDistribution, error) {
	if s.latest == nil {
		return nil, senderkeydistribution.ErrNotFound
	}
	dist, ok := s.latest[s.key(senderMemberID, receiverMemberID)]
	if !ok {
		return nil, senderkeydistribution.ErrNotFound
	}
	return dist, nil
}

func (s *senderKeyReqDistributionRepoStub) FindAvailableByRoomAndReceiver(context.Context, chatroom.ID, chatmember.ID) ([]*senderkeydistribution.SenderKeyDistribution, error) {
	return nil, nil
}

func (s *senderKeyReqDistributionRepoStub) FindByID(context.Context, senderkeydistribution.ID) (*senderkeydistribution.SenderKeyDistribution, error) {
	return nil, senderkeydistribution.ErrNotFound
}

func (s *senderKeyReqDistributionRepoStub) MarkConsumed(context.Context, senderkeydistribution.ID) error {
	return nil
}

func (s *senderKeyReqDistributionRepoStub) MarkFailed(context.Context, senderkeydistribution.ID) error {
	return nil
}

func (s *senderKeyReqDistributionRepoStub) key(senderMemberID, receiverMemberID chatmember.ID) string {
	return fmt.Sprintf("%d:%d", senderMemberID, receiverMemberID)
}

type senderKeyReqFriendshipStub struct {
	rows []*friendship.Friendship
	err  error
}

func (s *senderKeyReqFriendshipStub) FindBetweenUsers(context.Context, sharedDomain.UserID, sharedDomain.UserID) ([]*friendship.Friendship, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.rows, nil
}

func (s *senderKeyReqFriendshipStub) FindByUserID(context.Context, sharedDomain.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}

func (s *senderKeyReqFriendshipStub) FindAllByUserID(context.Context, sharedDomain.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}

func (s *senderKeyReqFriendshipStub) FindPendingByUserID(context.Context, sharedDomain.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}

func (s *senderKeyReqFriendshipStub) Create(context.Context, sharedDomain.UserID, sharedDomain.UserID) error {
	return nil
}

func (s *senderKeyReqFriendshipStub) CreateBlocked(context.Context, sharedDomain.UserID, sharedDomain.UserID) error {
	return nil
}

func (s *senderKeyReqFriendshipStub) FindByID(context.Context, int64) (*friendship.Friendship, error) {
	return nil, friendship.ErrFriendshipNotFound
}

func (s *senderKeyReqFriendshipStub) FindByUserIDAndFriendID(context.Context, sharedDomain.UserID, sharedDomain.UserID) (*friendship.Friendship, error) {
	return nil, friendship.ErrFriendshipNotFound
}

func (s *senderKeyReqFriendshipStub) FindPendingByFriendID(context.Context, sharedDomain.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}

func (s *senderKeyReqFriendshipStub) UpdateStatus(context.Context, int64, friendship.Status) error {
	return nil
}

func (s *senderKeyReqFriendshipStub) Delete(context.Context, int64) error {
	return nil
}

type senderKeyReqBroadcastCall struct {
	userID string
	msg    []byte
}

type senderKeyReqBroadcasterStub struct {
	mu    sync.Mutex
	calls []senderKeyReqBroadcastCall
}

func (s *senderKeyReqBroadcasterStub) SendToUser(userID string, msg []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, senderKeyReqBroadcastCall{userID: userID, msg: append([]byte(nil), msg...)})
}

func (s *senderKeyReqBroadcasterStub) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func (s *senderKeyReqBroadcasterStub) lastCall() senderKeyReqBroadcastCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[len(s.calls)-1]
}

var _ e2eePort.Broadcaster = (*senderKeyReqBroadcasterStub)(nil)

func makeCreateSKRInput(callerUserID int64, roomID, providerMemberID int64) appShared.UseCaseInput[CreateSenderKeyRequestInput] {
	uid := sharedDomain.UserID(callerUserID)
	auth := appShared.AuthContext{UserID: uid}
	return appShared.UseCaseInput[CreateSenderKeyRequestInput]{
		Base: appShared.BaseContext{Auth: &auth},
		Data: CreateSenderKeyRequestInput{RoomID: roomID, ProviderMemberID: providerMemberID},
	}
}

func makeParticipantStub(callerUID, providerUID int64) *senderKeyReqParticipantStub {
	callerPID := participant.ID(callerUID)
	providerPID := participant.ID(providerUID)
	callerUserID := sharedDomain.UserID(callerUID)
	providerUserID := sharedDomain.UserID(providerUID)
	return &senderKeyReqParticipantStub{
		byUserID: map[sharedDomain.UserID]*participant.Participant{
			callerUserID:   {ID: callerPID, Type: participant.UserType, UserID: &callerUserID},
			providerUserID: {ID: providerPID, Type: participant.UserType, UserID: &providerUserID},
		},
		byID: map[participant.ID]*participant.Participant{
			callerPID:   {ID: callerPID, Type: participant.UserType, UserID: &callerUserID},
			providerPID: {ID: providerPID, Type: participant.UserType, UserID: &providerUserID},
		},
	}
}

func makeChatMemberStub(roomID chatroom.ID, callerUID, providerUID int64, providerMemberRoomID chatroom.ID) (*senderKeyReqChatMemberStub, chatmember.ID, chatmember.ID) {
	callerMemberID := chatmember.ID(100 + callerUID)
	providerMemberID := chatmember.ID(100 + providerUID)
	callerPID := participant.ID(callerUID)
	providerPID := participant.ID(providerUID)
	stub := &senderKeyReqChatMemberStub{
		byID: map[chatmember.ID]*chatmember.ChatMember{
			callerMemberID:   {ID: callerMemberID, RoomID: roomID, ParticipantID: callerPID},
			providerMemberID: {ID: providerMemberID, RoomID: providerMemberRoomID, ParticipantID: providerPID},
		},
		byRoomAndParticipant: map[chatroom.ID]map[participant.ID]*chatmember.ChatMember{
			roomID: {
				callerPID: {ID: callerMemberID, RoomID: roomID, ParticipantID: callerPID},
			},
		},
	}
	return stub, callerMemberID, providerMemberID
}

func TestCreateSenderKeyRequest_ProviderInDifferentRoom(t *testing.T) {
	const (
		callerUID   = int64(1)
		providerUID = int64(2)
		roomID      = chatroom.ID(10)
		otherRoomID = chatroom.ID(99)
	)

	pStub := makeParticipantStub(callerUID, providerUID)
	cmStub, _, providerMemberID := makeChatMemberStub(roomID, callerUID, providerUID, otherRoomID)
	skrStub := &senderKeyReqSKRRepoStub{}
	mskStub := &senderKeyReqMSKRepoStub{findLatestErr: membersenderkey.ErrNotFound}

	uc := NewCreateSenderKeyRequestUseCase(
		pStub,
		cmStub,
		skrStub,
		mskStub,
		&senderKeyReqDistributionRepoStub{},
		&senderKeyReqFriendshipStub{},
		&senderKeyReqBroadcasterStub{},
	)
	input := makeCreateSKRInput(callerUID, int64(roomID), int64(providerMemberID))

	_, err := uc.Execute(context.Background(), input)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotRoomMember), "expected ErrNotRoomMember, got %v", err)
	assert.Zero(t, skrStub.storedCount(), "no request should be stored for cross-room provider")
}

func TestCreateSenderKeyRequest_CallerNotInRoom(t *testing.T) {
	const (
		callerUID   = int64(11)
		providerUID = int64(12)
		roomID      = chatroom.ID(110)
	)

	pStub := makeParticipantStub(callerUID, providerUID)
	cmStub, callerMemberID, providerMemberID := makeChatMemberStub(roomID, callerUID, providerUID, roomID)
	delete(cmStub.byRoomAndParticipant[roomID], participant.ID(callerUID))
	delete(cmStub.byID, callerMemberID)

	uc := NewCreateSenderKeyRequestUseCase(
		pStub,
		cmStub,
		&senderKeyReqSKRRepoStub{},
		&senderKeyReqMSKRepoStub{findLatestErr: membersenderkey.ErrNotFound},
		&senderKeyReqDistributionRepoStub{},
		&senderKeyReqFriendshipStub{},
		&senderKeyReqBroadcasterStub{},
	)

	_, err := uc.Execute(context.Background(), makeCreateSKRInput(callerUID, int64(roomID), int64(providerMemberID)))

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotRoomMember), "expected ErrNotRoomMember, got %v", err)
}

func TestCreateSenderKeyRequest_LatestDistributionAlreadyAvailable(t *testing.T) {
	const (
		callerUID   = int64(3)
		providerUID = int64(4)
		roomID      = chatroom.ID(20)
	)

	pStub := makeParticipantStub(callerUID, providerUID)
	cmStub, callerMemberID, providerMemberID := makeChatMemberStub(roomID, callerUID, providerUID, roomID)
	skrStub := &senderKeyReqSKRRepoStub{}
	mskStub := &senderKeyReqMSKRepoStub{}
	distStub := &senderKeyReqDistributionRepoStub{
		latest: map[string]*senderkeydistribution.SenderKeyDistribution{
			fmt.Sprintf("%d:%d", providerMemberID, callerMemberID): {
				SenderMemberID:   providerMemberID,
				ReceiverMemberID: callerMemberID,
				SenderKeyVersion: 9,
				Status:           senderkeydistribution.StatusAvailable,
			},
		},
	}

	uc := NewCreateSenderKeyRequestUseCase(
		pStub,
		cmStub,
		skrStub,
		mskStub,
		distStub,
		&senderKeyReqFriendshipStub{},
		&senderKeyReqBroadcasterStub{},
	)
	input := makeCreateSKRInput(callerUID, int64(roomID), int64(providerMemberID))

	_, err := uc.Execute(context.Background(), input)

	require.NoError(t, err)
	assert.Zero(t, skrStub.storedCount(), "no request should be stored when the latest distribution is already available")
}

func TestCreateSenderKeyRequest_BlockedRelationship(t *testing.T) {
	const (
		callerUID   = int64(5)
		providerUID = int64(6)
		roomID      = chatroom.ID(30)
	)

	callerUID64 := sharedDomain.UserID(callerUID)
	providerUID64 := sharedDomain.UserID(providerUID)
	blockedRow := &friendship.Friendship{Status: friendship.StatusBlocked, UserID: callerUID64, FriendID: providerUID64}

	pStub := makeParticipantStub(callerUID, providerUID)
	cmStub, _, providerMemberID := makeChatMemberStub(roomID, callerUID, providerUID, roomID)
	skrStub := &senderKeyReqSKRRepoStub{}
	mskStub := &senderKeyReqMSKRepoStub{findLatestErr: membersenderkey.ErrNotFound}
	fsStub := &senderKeyReqFriendshipStub{rows: []*friendship.Friendship{blockedRow}}

	uc := NewCreateSenderKeyRequestUseCase(
		pStub,
		cmStub,
		skrStub,
		mskStub,
		&senderKeyReqDistributionRepoStub{},
		fsStub,
		&senderKeyReqBroadcasterStub{},
	)
	input := makeCreateSKRInput(callerUID, int64(roomID), int64(providerMemberID))

	_, err := uc.Execute(context.Background(), input)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrForbidden), "expected ErrForbidden for blocked relationship, got %v", err)
	assert.Zero(t, skrStub.storedCount())
}

func TestCreateSenderKeyRequest_Success(t *testing.T) {
	const (
		callerUID   = int64(7)
		providerUID = int64(8)
		roomID      = chatroom.ID(40)
	)

	pStub := makeParticipantStub(callerUID, providerUID)
	cmStub, callerMemberID, providerMemberID := makeChatMemberStub(roomID, callerUID, providerUID, roomID)
	skrStub := &senderKeyReqSKRRepoStub{}
	mskStub := &senderKeyReqMSKRepoStub{findLatestErr: membersenderkey.ErrNotFound}
	broadcaster := &senderKeyReqBroadcasterStub{}

	uc := NewCreateSenderKeyRequestUseCase(
		pStub,
		cmStub,
		skrStub,
		mskStub,
		&senderKeyReqDistributionRepoStub{},
		&senderKeyReqFriendshipStub{err: friendship.ErrFriendshipNotFound},
		broadcaster,
	)
	input := makeCreateSKRInput(callerUID, int64(roomID), int64(providerMemberID))

	_, err := uc.Execute(context.Background(), input)

	require.NoError(t, err)
	require.Equal(t, 1, skrStub.storedCount())
	stored := skrStub.storedRequest(callerMemberID, providerMemberID)
	require.NotNil(t, stored)
	assert.Equal(t, callerMemberID, stored.RequesterMemberID)
	assert.Equal(t, providerMemberID, stored.ProviderMemberID)

	require.Eventually(t, func() bool {
		return broadcaster.callCount() == 1
	}, time.Second, 10*time.Millisecond)

	call := broadcaster.lastCall()
	assert.Equal(t, fmt.Sprint(providerUID), call.userID)

	var msg struct {
		Type    string `json:"type"`
		Payload struct {
			RoomID            int64 `json:"room_id"`
			ProviderMemberID  int64 `json:"provider_member_id"`
			RequesterMemberID int64 `json:"requester_member_id"`
			RequesterUserID   int64 `json:"requester_user_id"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(call.msg, &msg))
	assert.Equal(t, "e2ee.sender_key_needed", msg.Type)
	assert.Equal(t, int64(roomID), msg.Payload.RoomID)
	assert.Equal(t, int64(providerMemberID), msg.Payload.ProviderMemberID)
	assert.Equal(t, int64(callerMemberID), msg.Payload.RequesterMemberID)
	assert.Equal(t, callerUID, msg.Payload.RequesterUserID)
}

func TestCreateSenderKeyRequest_RepeatedRequestsUseUpsertContract(t *testing.T) {
	const (
		callerUID   = int64(13)
		providerUID = int64(14)
		roomID      = chatroom.ID(140)
	)

	pStub := makeParticipantStub(callerUID, providerUID)
	cmStub, callerMemberID, providerMemberID := makeChatMemberStub(roomID, callerUID, providerUID, roomID)
	skrStub := &senderKeyReqSKRRepoStub{}
	mskStub := &senderKeyReqMSKRepoStub{findLatestErr: membersenderkey.ErrNotFound}

	uc := NewCreateSenderKeyRequestUseCase(
		pStub,
		cmStub,
		skrStub,
		mskStub,
		&senderKeyReqDistributionRepoStub{},
		&senderKeyReqFriendshipStub{err: friendship.ErrFriendshipNotFound},
		&senderKeyReqBroadcasterStub{},
	)
	input := makeCreateSKRInput(callerUID, int64(roomID), int64(providerMemberID))

	_, err := uc.Execute(context.Background(), input)
	require.NoError(t, err)

	_, err = uc.Execute(context.Background(), input)
	require.NoError(t, err)

	assert.Equal(t, 2, skrStub.upsertCount)
	assert.Equal(t, 1, skrStub.storedCount())
	stored := skrStub.storedRequest(callerMemberID, providerMemberID)
	require.NotNil(t, stored)
	assert.Equal(t, callerMemberID, stored.RequesterMemberID)
	assert.Equal(t, providerMemberID, stored.ProviderMemberID)
}

package usecase

import (
	"context"
	"testing"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyreceipt"
	sharedDomain "github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	domainuser "github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	senderStatusRoomID            = int64(4)
	senderStatusCallerUserID      = int64(100)
	senderStatusPeerUserID        = int64(200)
	senderStatusCallerAccountID   = sharedDomain.AccountID(1000)
	senderStatusPeerAccountID     = sharedDomain.AccountID(2000)
	senderStatusCallerParticipant = participant.ID(1)
	senderStatusPeerParticipant   = participant.ID(2)
)

type senderStatusParticipantRepoStub struct {
	byUserID map[sharedDomain.UserID]*participant.Participant
	byID     map[participant.ID]*participant.Participant
}

func (s *senderStatusParticipantRepoStub) FindByID(_ context.Context, participantID participant.ID) (*participant.Participant, error) {
	p, ok := s.byID[participantID]
	if !ok {
		return nil, participant.ErrNotFound
	}
	return p, nil
}

func (s *senderStatusParticipantRepoStub) FindByUserID(_ context.Context, userID sharedDomain.UserID) (*participant.Participant, error) {
	p, ok := s.byUserID[userID]
	if !ok {
		return nil, participant.ErrNotFound
	}
	return p, nil
}

func (*senderStatusParticipantRepoStub) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (*senderStatusParticipantRepoStub) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (*senderStatusParticipantRepoStub) Create(context.Context, *participant.Participant) error {
	return nil
}

type senderStatusChatMemberRepoStub struct {
	byID                 map[chatmember.ID]*chatmember.ChatMember
	byRoomAndParticipant map[chatroom.ID]map[participant.ID]*chatmember.ChatMember
	byRoom               map[chatroom.ID][]*chatmember.ChatMember
}

func (s *senderStatusChatMemberRepoStub) FindByID(_ context.Context, id chatmember.ID) (*chatmember.ChatMember, error) {
	member, ok := s.byID[id]
	if !ok {
		return nil, chatmember.ErrNotFound
	}
	return member, nil
}

func (s *senderStatusChatMemberRepoStub) FindByRoomAndParticipant(_ context.Context, roomID chatroom.ID, participantID participant.ID) (*chatmember.ChatMember, error) {
	if roomMembers, ok := s.byRoomAndParticipant[roomID]; ok {
		if member, exists := roomMembers[participantID]; exists {
			return member, nil
		}
	}
	return nil, chatmember.ErrNotFound
}

func (s *senderStatusChatMemberRepoStub) FindByRoom(_ context.Context, roomID chatroom.ID) ([]*chatmember.ChatMember, error) {
	return s.byRoom[roomID], nil
}

func (*senderStatusChatMemberRepoStub) FindByParticipant(context.Context, participant.ID) ([]*chatmember.ChatMember, error) {
	return nil, nil
}

func (*senderStatusChatMemberRepoStub) Add(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (*senderStatusChatMemberRepoStub) Update(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (*senderStatusChatMemberRepoStub) SoftDelete(context.Context, chatmember.ID) error {
	return nil
}

func (*senderStatusChatMemberRepoStub) Remove(context.Context, chatroom.ID, participant.ID) error {
	return nil
}

type senderStatusMemberSenderKeyRepoStub struct {
	findLatest func(ctx context.Context, chatMemberID chatmember.ID) (*membersenderkey.MemberSenderKey, error)
}

func (s *senderStatusMemberSenderKeyRepoStub) FindLatest(ctx context.Context, chatMemberID chatmember.ID) (*membersenderkey.MemberSenderKey, error) {
	if s.findLatest != nil {
		return s.findLatest(ctx, chatMemberID)
	}
	return nil, membersenderkey.ErrNotFound
}

func (*senderStatusMemberSenderKeyRepoStub) FindAllByMembers(context.Context, []chatmember.ID) ([]*membersenderkey.MemberSenderKey, error) {
	return nil, nil
}

func (*senderStatusMemberSenderKeyRepoStub) Add(context.Context, *membersenderkey.MemberSenderKey) error {
	return nil
}

func (*senderStatusMemberSenderKeyRepoStub) UpsertLatest(context.Context, *membersenderkey.MemberSenderKey) error {
	return nil
}

type senderStatusDistributionRepoStub struct {
	findLatestForReceiver func(ctx context.Context, senderMemberID chatmember.ID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) (*senderkeydistribution.SenderKeyDistribution, error)
}

func (*senderStatusDistributionRepoStub) UpsertBatch(context.Context, []*senderkeydistribution.SenderKeyDistribution) error {
	return nil
}

func (*senderStatusDistributionRepoStub) FindPendingReceivers(context.Context, chatmember.ID, sharedDomain.DeviceID, int64) ([]chatmember.ID, error) {
	return []chatmember.ID{}, nil
}

func (*senderStatusDistributionRepoStub) UpsertAvailable(context.Context, *senderkeydistribution.SenderKeyDistribution) error {
	return nil
}

func (*senderStatusDistributionRepoStub) FindLatest(context.Context, chatmember.ID, sharedDomain.DeviceID, chatmember.ID, sharedDomain.DeviceID) (*senderkeydistribution.SenderKeyDistribution, error) {
	return nil, senderkeydistribution.ErrNotFound
}

func (s *senderStatusDistributionRepoStub) FindLatestForReceiver(ctx context.Context, senderMemberID chatmember.ID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) (*senderkeydistribution.SenderKeyDistribution, error) {
	if s.findLatestForReceiver != nil {
		return s.findLatestForReceiver(ctx, senderMemberID, receiverMemberID, receiverDeviceID)
	}
	return nil, senderkeydistribution.ErrNotFound
}

func (*senderStatusDistributionRepoStub) FindAvailableByRoomAndReceiver(context.Context, chatroom.ID, chatmember.ID, sharedDomain.DeviceID) ([]*senderkeydistribution.SenderKeyDistribution, error) {
	return nil, nil
}

func (*senderStatusDistributionRepoStub) FindByID(context.Context, senderkeydistribution.ID) (*senderkeydistribution.SenderKeyDistribution, error) {
	return nil, senderkeydistribution.ErrNotFound
}

func (*senderStatusDistributionRepoStub) MarkConsumed(context.Context, senderkeydistribution.ID) error {
	return nil
}

func (*senderStatusDistributionRepoStub) MarkFailed(context.Context, senderkeydistribution.ID) error {
	return nil
}

type senderStatusReceiptRepoStub struct {
	findLatest func(ctx context.Context, senderMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) (*senderkeyreceipt.SenderKeyReceipt, error)
}

func (s *senderStatusReceiptRepoStub) FindLatest(ctx context.Context, senderMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) (*senderkeyreceipt.SenderKeyReceipt, error) {
	if s.findLatest != nil {
		return s.findLatest(ctx, senderMemberID, receiverDeviceID)
	}
	return nil, senderkeyreceipt.ErrNotFound
}

func (*senderStatusReceiptRepoStub) Upsert(context.Context, *senderkeyreceipt.SenderKeyReceipt) error {
	return nil
}

type senderStatusAccountRepoStub struct {
	byID map[sharedDomain.AccountID]*account.Account
}

func (s *senderStatusAccountRepoStub) FindByID(_ context.Context, accountID sharedDomain.AccountID) (*account.Account, error) {
	if acc, ok := s.byID[accountID]; ok {
		return acc, nil
	}
	return nil, account.ErrAccountNotFound
}

func (*senderStatusAccountRepoStub) FindByAccountName(context.Context, string) (*account.Account, error) {
	return nil, account.ErrAccountNotFound
}

func (*senderStatusAccountRepoStub) FindByEmail(context.Context, sharedDomain.EmailAddress) (*account.Account, error) {
	return nil, account.ErrAccountNotFound
}

func (*senderStatusAccountRepoStub) Create(context.Context, *account.Account) (sharedDomain.AccountID, error) {
	return 0, nil
}

func (*senderStatusAccountRepoStub) Update(context.Context, *account.Account) error { return nil }

func (*senderStatusAccountRepoStub) RegisterDevice(context.Context, *account.AccountDevice) error {
	return nil
}

func (*senderStatusAccountRepoStub) UpdateDeviceStatus(context.Context, sharedDomain.AccountID, sharedDomain.DeviceID, account.DeviceStatus) error {
	return nil
}

func (*senderStatusAccountRepoStub) RecordLoginEvent(context.Context, *account.AccountLoginEvent) error {
	return nil
}

func (*senderStatusAccountRepoStub) ReplaceDevices(context.Context, sharedDomain.AccountID, []account.AccountDevice) error {
	return nil
}

type senderStatusUserRepoStub struct {
	byID map[sharedDomain.UserID]*domainuser.User
}

func (*senderStatusUserRepoStub) Create(context.Context, *domainuser.User) (sharedDomain.UserID, error) {
	return 0, nil
}

func (s *senderStatusUserRepoStub) FindByID(_ context.Context, userID sharedDomain.UserID) (*domainuser.User, error) {
	if user, ok := s.byID[userID]; ok {
		return user, nil
	}
	return nil, domainuser.ErrUserNotFound
}

func (*senderStatusUserRepoStub) FindByAccountID(context.Context, sharedDomain.AccountID) (*[]domainuser.User, error) {
	return nil, nil
}

func (*senderStatusUserRepoStub) Update(context.Context, *domainuser.User) error { return nil }

func (*senderStatusUserRepoStub) SearchByName(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}

func (*senderStatusUserRepoStub) FindByAccountName(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}

func (*senderStatusUserRepoStub) FindByPublicID(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}

func makeSenderStatusIdentityStubs(roomID chatroom.ID, callerMemberID, peerMemberID chatmember.ID) (
	*senderStatusParticipantRepoStub,
	*senderStatusChatMemberRepoStub,
	*senderStatusAccountRepoStub,
	*senderStatusUserRepoStub,
) {
	callerUserID := sharedDomain.UserID(senderStatusCallerUserID)
	peerUserID := sharedDomain.UserID(senderStatusPeerUserID)
	callerParticipant := &participant.Participant{ID: senderStatusCallerParticipant, Type: participant.UserType, UserID: &callerUserID}
	peerParticipant := &participant.Participant{ID: senderStatusPeerParticipant, Type: participant.UserType, UserID: &peerUserID}
	callerMember := &chatmember.ChatMember{ID: callerMemberID, RoomID: roomID, ParticipantID: senderStatusCallerParticipant}
	peerMember := &chatmember.ChatMember{ID: peerMemberID, RoomID: roomID, ParticipantID: senderStatusPeerParticipant}

	return &senderStatusParticipantRepoStub{
			byUserID: map[sharedDomain.UserID]*participant.Participant{
				callerUserID: callerParticipant,
				peerUserID:   peerParticipant,
			},
			byID: map[participant.ID]*participant.Participant{
				senderStatusCallerParticipant: callerParticipant,
				senderStatusPeerParticipant:   peerParticipant,
			},
		},
		&senderStatusChatMemberRepoStub{
			byID: map[chatmember.ID]*chatmember.ChatMember{
				callerMemberID: callerMember,
				peerMemberID:   peerMember,
			},
			byRoomAndParticipant: map[chatroom.ID]map[participant.ID]*chatmember.ChatMember{
				roomID: {
					senderStatusCallerParticipant: callerMember,
					senderStatusPeerParticipant:   peerMember,
				},
			},
			byRoom: map[chatroom.ID][]*chatmember.ChatMember{
				roomID: {callerMember, peerMember},
			},
		},
		&senderStatusAccountRepoStub{
			byID: map[sharedDomain.AccountID]*account.Account{
				senderStatusCallerAccountID: {
					ID: senderStatusCallerAccountID,
					Devices: []account.AccountDevice{{
						AccountID: senderStatusCallerAccountID,
						DeviceID:  senderKeyReqRequesterDeviceID,
						Status:    account.DeviceStatusReady,
					}},
				},
				senderStatusPeerAccountID: {
					ID: senderStatusPeerAccountID,
					Devices: []account.AccountDevice{{
						AccountID: senderStatusPeerAccountID,
						DeviceID:  senderKeyReqProviderDeviceID,
						Status:    account.DeviceStatusReady,
					}},
				},
			},
		},
		&senderStatusUserRepoStub{
			byID: map[sharedDomain.UserID]*domainuser.User{
				callerUserID: {ID: callerUserID, AccountID: senderStatusCallerAccountID},
				peerUserID:   {ID: peerUserID, AccountID: senderStatusPeerAccountID},
			},
		}
}

func makeSenderStatusInput() appShared.UseCaseInput[GetSenderKeyDistributionStatusInput] {
	return appShared.UseCaseInput[GetSenderKeyDistributionStatusInput]{
		Base: appShared.BaseContext{
			Auth:    &appShared.AuthContext{UserID: sharedDomain.UserID(senderStatusCallerUserID)},
			Request: appShared.RequestContext{DeviceID: senderKeyReqRequesterDeviceID},
		},
		Data: GetSenderKeyDistributionStatusInput{RoomID: senderStatusRoomID},
	}
}

func peerRouteRef(peerMemberID chatmember.ID) SenderKeyRouteRef {
	return SenderKeyRouteRef{
		UserID:   senderStatusPeerUserID,
		MemberID: int64(peerMemberID),
		DeviceID: senderKeyReqProviderDeviceID.String(),
	}
}

func TestGetSenderKeyDistributionStatusUseCase_EmptySlicesWhenNoKeys(t *testing.T) {
	callerMemberID := chatmember.ID(10)
	peerMemberID := chatmember.ID(11)
	participantRepo, chatMemberRepo, accountRepo, userRepo := makeSenderStatusIdentityStubs(chatroom.ID(senderStatusRoomID), callerMemberID, peerMemberID)

	uc := NewGetSenderKeyDistributionStatusUseCase(
		participantRepo,
		chatMemberRepo,
		accountRepo,
		userRepo,
		&senderStatusMemberSenderKeyRepoStub{},
		&senderStatusDistributionRepoStub{},
		&senderStatusReceiptRepoStub{},
	)

	out, err := uc.Execute(context.Background(), makeSenderStatusInput())

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.False(t, out.OwnMemberSenderKeyExists)
	assert.Empty(t, out.RequestableSources)
	assert.Empty(t, out.AvailableFromSources)
	assert.Empty(t, out.AvailableToTargets)
	assert.Empty(t, out.PendingReceivers)
	assert.Empty(t, out.PendingFromSources)
}

func TestGetSenderKeyDistributionStatusUseCase_PendingFromMembers(t *testing.T) {
	callerMemberID := chatmember.ID(10)
	peerMemberID := chatmember.ID(11)
	participantRepo, chatMemberRepo, accountRepo, userRepo := makeSenderStatusIdentityStubs(chatroom.ID(senderStatusRoomID), callerMemberID, peerMemberID)

	uc := NewGetSenderKeyDistributionStatusUseCase(
		participantRepo,
		chatMemberRepo,
		accountRepo,
		userRepo,
		&senderStatusMemberSenderKeyRepoStub{
			findLatest: func(_ context.Context, chatMemberID chatmember.ID) (*membersenderkey.MemberSenderKey, error) {
				if chatMemberID == peerMemberID {
					return &membersenderkey.MemberSenderKey{
						ChatMemberID:     peerMemberID,
						SenderDeviceID:   senderKeyReqProviderDeviceID,
						SenderKeyVersion: 7,
						ChainID:          membersenderkey.ChainID(7),
					}, nil
				}
				return nil, membersenderkey.ErrNotFound
			},
		},
		&senderStatusDistributionRepoStub{},
		&senderStatusReceiptRepoStub{},
	)

	out, err := uc.Execute(context.Background(), makeSenderStatusInput())

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.False(t, out.OwnMemberSenderKeyExists)
	assert.Equal(t, []SenderKeyRouteRef{peerRouteRef(peerMemberID)}, out.RequestableSources)
	assert.Empty(t, out.AvailableFromSources)
	assert.Empty(t, out.AvailableToTargets)
	assert.Equal(t, []SenderKeyRouteRef{peerRouteRef(peerMemberID)}, out.PendingFromSources)
}

func TestGetSenderKeyDistributionStatusUseCase_OwnSenderKeyExists(t *testing.T) {
	callerMemberID := chatmember.ID(10)
	peerMemberID := chatmember.ID(11)
	participantRepo, chatMemberRepo, accountRepo, userRepo := makeSenderStatusIdentityStubs(chatroom.ID(senderStatusRoomID), callerMemberID, peerMemberID)

	uc := NewGetSenderKeyDistributionStatusUseCase(
		participantRepo,
		chatMemberRepo,
		accountRepo,
		userRepo,
		&senderStatusMemberSenderKeyRepoStub{
			findLatest: func(_ context.Context, chatMemberID chatmember.ID) (*membersenderkey.MemberSenderKey, error) {
				switch chatMemberID {
				case callerMemberID:
					return &membersenderkey.MemberSenderKey{
						ChatMemberID:     callerMemberID,
						SenderDeviceID:   senderKeyReqRequesterDeviceID,
						SenderKeyVersion: 7,
						ChainID:          membersenderkey.ChainID(7),
					}, nil
				case peerMemberID:
					return &membersenderkey.MemberSenderKey{
						ChatMemberID:     peerMemberID,
						SenderDeviceID:   senderKeyReqProviderDeviceID,
						SenderKeyVersion: 5,
						ChainID:          membersenderkey.ChainID(5),
					}, nil
				default:
					return nil, membersenderkey.ErrNotFound
				}
			},
		},
		&senderStatusDistributionRepoStub{
			findLatestForReceiver: func(_ context.Context, senderMemberID chatmember.ID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) (*senderkeydistribution.SenderKeyDistribution, error) {
				switch {
				case senderMemberID == peerMemberID && receiverMemberID == callerMemberID && receiverDeviceID == senderKeyReqRequesterDeviceID:
					return &senderkeydistribution.SenderKeyDistribution{
						SenderMemberID:   peerMemberID,
						SenderDeviceID:   senderKeyReqProviderDeviceID,
						ReceiverMemberID: callerMemberID,
						ReceiverDeviceID: senderKeyReqRequesterDeviceID,
						SenderKeyVersion: 5,
						Status:           senderkeydistribution.StatusAvailable,
					}, nil
				case senderMemberID == callerMemberID && receiverMemberID == peerMemberID && receiverDeviceID == senderKeyReqProviderDeviceID:
					return &senderkeydistribution.SenderKeyDistribution{
						SenderMemberID:   callerMemberID,
						SenderDeviceID:   senderKeyReqRequesterDeviceID,
						ReceiverMemberID: peerMemberID,
						ReceiverDeviceID: senderKeyReqProviderDeviceID,
						SenderKeyVersion: 7,
						Status:           senderkeydistribution.StatusConsumed,
					}, nil
				default:
					return nil, senderkeydistribution.ErrNotFound
				}
			},
		},
		&senderStatusReceiptRepoStub{},
	)

	out, err := uc.Execute(context.Background(), makeSenderStatusInput())

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, out.OwnMemberSenderKeyExists)
	assert.Empty(t, out.RequestableSources)
	assert.Equal(t, []SenderKeyRouteRef{peerRouteRef(peerMemberID)}, out.AvailableFromSources)
	assert.Empty(t, out.PendingReceivers)
	assert.Empty(t, out.AvailableToTargets)
	assert.Empty(t, out.PendingFromSources)
}

func TestGetSenderKeyDistributionStatusUseCase_PendingReceiversRequiresFreshUpload(t *testing.T) {
	callerMemberID := chatmember.ID(10)
	peerMemberID := chatmember.ID(11)
	participantRepo, chatMemberRepo, accountRepo, userRepo := makeSenderStatusIdentityStubs(chatroom.ID(senderStatusRoomID), callerMemberID, peerMemberID)

	uc := NewGetSenderKeyDistributionStatusUseCase(
		participantRepo,
		chatMemberRepo,
		accountRepo,
		userRepo,
		&senderStatusMemberSenderKeyRepoStub{
			findLatest: func(_ context.Context, chatMemberID chatmember.ID) (*membersenderkey.MemberSenderKey, error) {
				switch chatMemberID {
				case callerMemberID:
					return &membersenderkey.MemberSenderKey{
						ChatMemberID:     callerMemberID,
						SenderDeviceID:   senderKeyReqRequesterDeviceID,
						SenderKeyVersion: 9,
						ChainID:          membersenderkey.ChainID(9),
					}, nil
				case peerMemberID:
					return &membersenderkey.MemberSenderKey{
						ChatMemberID:     peerMemberID,
						SenderDeviceID:   senderKeyReqProviderDeviceID,
						SenderKeyVersion: 5,
						ChainID:          membersenderkey.ChainID(5),
					}, nil
				default:
					return nil, membersenderkey.ErrNotFound
				}
			},
		},
		&senderStatusDistributionRepoStub{
			findLatestForReceiver: func(_ context.Context, senderMemberID chatmember.ID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) (*senderkeydistribution.SenderKeyDistribution, error) {
				switch {
				case senderMemberID == peerMemberID && receiverMemberID == callerMemberID && receiverDeviceID == senderKeyReqRequesterDeviceID:
					return &senderkeydistribution.SenderKeyDistribution{
						SenderMemberID:   peerMemberID,
						SenderDeviceID:   senderKeyReqProviderDeviceID,
						ReceiverMemberID: callerMemberID,
						ReceiverDeviceID: senderKeyReqRequesterDeviceID,
						SenderKeyVersion: 5,
						Status:           senderkeydistribution.StatusConsumed,
					}, nil
				case senderMemberID == callerMemberID && receiverMemberID == peerMemberID && receiverDeviceID == senderKeyReqProviderDeviceID:
					return &senderkeydistribution.SenderKeyDistribution{
						SenderMemberID:   callerMemberID,
						SenderDeviceID:   senderKeyReqRequesterDeviceID,
						ReceiverMemberID: peerMemberID,
						ReceiverDeviceID: senderKeyReqProviderDeviceID,
						SenderKeyVersion: 9,
						Status:           senderkeydistribution.StatusFailed,
					}, nil
				default:
					return nil, senderkeydistribution.ErrNotFound
				}
			},
		},
		&senderStatusReceiptRepoStub{},
	)

	out, err := uc.Execute(context.Background(), makeSenderStatusInput())

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, out.OwnMemberSenderKeyExists)
	assert.Empty(t, out.AvailableToTargets)
	assert.Equal(t, []SenderKeyRouteRef{peerRouteRef(peerMemberID)}, out.PendingReceivers)
}

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

type senderStatusParticipantRepoStub struct {
	findByUserID func(ctx context.Context, userID sharedDomain.UserID) (*participant.Participant, error)
	findByID     func(ctx context.Context, participantID participant.ID) (*participant.Participant, error)
}

func (s *senderStatusParticipantRepoStub) FindByID(ctx context.Context, participantID participant.ID) (*participant.Participant, error) {
	if s.findByID != nil {
		return s.findByID(ctx, participantID)
	}
	uid := sharedDomain.UserID(1)
	return &participant.Participant{ID: participantID, Type: participant.UserType, UserID: &uid}, nil
}

func (s *senderStatusParticipantRepoStub) FindByUserID(ctx context.Context, userID sharedDomain.UserID) (*participant.Participant, error) {
	if s.findByUserID != nil {
		return s.findByUserID(ctx, userID)
	}
	return nil, participant.ErrNotFound
}

func (s *senderStatusParticipantRepoStub) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *senderStatusParticipantRepoStub) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *senderStatusParticipantRepoStub) Create(context.Context, *participant.Participant) error {
	return nil
}

type senderStatusChatMemberRepoStub struct {
	findByRoomAndParticipant func(ctx context.Context, roomID chatroom.ID, participantID participant.ID) (*chatmember.ChatMember, error)
	findByRoom               func(ctx context.Context, roomID chatroom.ID) ([]*chatmember.ChatMember, error)
}

func (s *senderStatusChatMemberRepoStub) FindByID(context.Context, chatmember.ID) (*chatmember.ChatMember, error) {
	return nil, chatmember.ErrNotFound
}

func (s *senderStatusChatMemberRepoStub) FindByRoomAndParticipant(ctx context.Context, roomID chatroom.ID, participantID participant.ID) (*chatmember.ChatMember, error) {
	if s.findByRoomAndParticipant != nil {
		return s.findByRoomAndParticipant(ctx, roomID, participantID)
	}
	return nil, chatmember.ErrNotFound
}

func (s *senderStatusChatMemberRepoStub) FindByRoom(ctx context.Context, roomID chatroom.ID) ([]*chatmember.ChatMember, error) {
	if s.findByRoom != nil {
		return s.findByRoom(ctx, roomID)
	}
	return nil, nil
}

func (s *senderStatusChatMemberRepoStub) FindByParticipant(context.Context, participant.ID) ([]*chatmember.ChatMember, error) {
	return nil, nil
}

func (s *senderStatusChatMemberRepoStub) Add(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (s *senderStatusChatMemberRepoStub) Update(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (s *senderStatusChatMemberRepoStub) SoftDelete(context.Context, chatmember.ID) error {
	return nil
}

func (s *senderStatusChatMemberRepoStub) Remove(context.Context, chatroom.ID, participant.ID) error {
	return nil
}

type senderStatusMemberSenderKeyRepoStub struct {
	findLatestForMember func(ctx context.Context, chatMemberID chatmember.ID) ([]*membersenderkey.MemberSenderKey, error)
}

func (s *senderStatusMemberSenderKeyRepoStub) FindLatest(ctx context.Context, chatMemberID chatmember.ID, senderDeviceID sharedDomain.DeviceID) (*membersenderkey.MemberSenderKey, error) {
	keys, err := s.FindLatestForMember(ctx, chatMemberID)
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		if key.SenderDeviceID == senderDeviceID {
			copied := *key
			return &copied, nil
		}
	}
	return nil, membersenderkey.ErrNotFound
}

func (s *senderStatusMemberSenderKeyRepoStub) FindLatestForMember(ctx context.Context, chatMemberID chatmember.ID) ([]*membersenderkey.MemberSenderKey, error) {
	if s.findLatestForMember != nil {
		return s.findLatestForMember(ctx, chatMemberID)
	}
	return nil, membersenderkey.ErrNotFound
}

func (s *senderStatusMemberSenderKeyRepoStub) FindAllByMembers(context.Context, []chatmember.ID) ([]*membersenderkey.MemberSenderKey, error) {
	return nil, nil
}

func (s *senderStatusMemberSenderKeyRepoStub) Add(context.Context, *membersenderkey.MemberSenderKey) error {
	return nil
}

func (s *senderStatusMemberSenderKeyRepoStub) UpsertLatest(context.Context, *membersenderkey.MemberSenderKey) error {
	return nil
}

type senderStatusDistributionRepoStub struct {
	findLatest                     func(ctx context.Context, senderMemberID chatmember.ID, senderDeviceID sharedDomain.DeviceID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) (*senderkeydistribution.SenderKeyDistribution, error)
	findAvailableByRoomAndReceiver func(ctx context.Context, roomID chatroom.ID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) ([]*senderkeydistribution.SenderKeyDistribution, error)
}

func (s *senderStatusDistributionRepoStub) UpsertBatch(context.Context, []*senderkeydistribution.SenderKeyDistribution) error {
	return nil
}

func (s *senderStatusDistributionRepoStub) FindPendingReceivers(ctx context.Context, senderMemberID chatmember.ID, senderDeviceID sharedDomain.DeviceID, latestChainID int64) ([]chatmember.ID, error) {
	_, _, _ = ctx, senderMemberID, senderDeviceID
	_ = latestChainID
	return []chatmember.ID{}, nil
}

func (s *senderStatusDistributionRepoStub) UpsertAvailable(context.Context, *senderkeydistribution.SenderKeyDistribution) error {
	return nil
}

func (s *senderStatusDistributionRepoStub) FindLatest(ctx context.Context, senderMemberID chatmember.ID, senderDeviceID sharedDomain.DeviceID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) (*senderkeydistribution.SenderKeyDistribution, error) {
	if s.findLatest != nil {
		return s.findLatest(ctx, senderMemberID, senderDeviceID, receiverMemberID, receiverDeviceID)
	}
	return nil, senderkeydistribution.ErrNotFound
}

func (s *senderStatusDistributionRepoStub) FindAvailableByRoomAndReceiver(ctx context.Context, roomID chatroom.ID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) ([]*senderkeydistribution.SenderKeyDistribution, error) {
	if s.findAvailableByRoomAndReceiver != nil {
		return s.findAvailableByRoomAndReceiver(ctx, roomID, receiverMemberID, receiverDeviceID)
	}
	return nil, nil
}

func (s *senderStatusDistributionRepoStub) FindByID(context.Context, senderkeydistribution.ID) (*senderkeydistribution.SenderKeyDistribution, error) {
	return nil, senderkeydistribution.ErrNotFound
}

func (s *senderStatusDistributionRepoStub) MarkConsumed(context.Context, senderkeydistribution.ID) error {
	return nil
}

func (s *senderStatusDistributionRepoStub) MarkFailed(context.Context, senderkeydistribution.ID) error {
	return nil
}

type senderStatusReceiptRepoStub struct {
	findLatest func(ctx context.Context, senderMemberID chatmember.ID, senderDeviceID sharedDomain.DeviceID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) (*senderkeyreceipt.SenderKeyReceipt, error)
}

func (s *senderStatusReceiptRepoStub) FindLatest(ctx context.Context, senderMemberID chatmember.ID, senderDeviceID sharedDomain.DeviceID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) (*senderkeyreceipt.SenderKeyReceipt, error) {
	if s.findLatest != nil {
		return s.findLatest(ctx, senderMemberID, senderDeviceID, receiverMemberID, receiverDeviceID)
	}
	return nil, senderkeyreceipt.ErrNotFound
}

func (s *senderStatusReceiptRepoStub) Upsert(context.Context, *senderkeyreceipt.SenderKeyReceipt) error {
	return nil
}

type senderStatusAccountRepoStub struct{}

func (*senderStatusAccountRepoStub) FindByID(context.Context, sharedDomain.AccountID) (*account.Account, error) {
	deviceID, _ := sharedDomain.ParseDeviceID("11111111-1111-1111-1111-111111111111")
	return &account.Account{
		ID: 1,
		Devices: []account.AccountDevice{{
			AccountID: 1,
			DeviceID:  deviceID,
			Status:    account.DeviceStatusReady,
		}},
	}, nil
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

type senderStatusUserRepoStub struct{}

func (*senderStatusUserRepoStub) Create(context.Context, *domainuser.User) (sharedDomain.UserID, error) {
	return 0, nil
}
func (*senderStatusUserRepoStub) FindByID(context.Context, sharedDomain.UserID) (*domainuser.User, error) {
	return &domainuser.User{ID: 1, AccountID: 1}, nil
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

func TestGetSenderKeyDistributionStatusUseCase_EmptySlicesWhenNoKeys(t *testing.T) {
	uc := NewGetSenderKeyDistributionStatusUseCase(
		&senderStatusParticipantRepoStub{
			findByUserID: func(context.Context, sharedDomain.UserID) (*participant.Participant, error) {
				return &participant.Participant{ID: participant.ID(1)}, nil
			},
		},
		&senderStatusChatMemberRepoStub{
			findByRoomAndParticipant: func(context.Context, chatroom.ID, participant.ID) (*chatmember.ChatMember, error) {
				return &chatmember.ChatMember{ID: chatmember.ID(10)}, nil
			},
			findByRoom: func(context.Context, chatroom.ID) ([]*chatmember.ChatMember, error) {
				return []*chatmember.ChatMember{
					{ID: chatmember.ID(10)},
					{ID: chatmember.ID(11)},
				}, nil
			},
		},
		&senderStatusAccountRepoStub{},
		&senderStatusUserRepoStub{},
		&senderStatusMemberSenderKeyRepoStub{
			findLatestForMember: func(_ context.Context, chatMemberID chatmember.ID) ([]*membersenderkey.MemberSenderKey, error) {
				_ = chatMemberID
				return nil, membersenderkey.ErrNotFound
			},
		},
		&senderStatusDistributionRepoStub{},
		&senderStatusReceiptRepoStub{},
	)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[GetSenderKeyDistributionStatusInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{UserID: sharedDomain.UserID(100)},
			Request: appShared.RequestContext{DeviceID: senderKeyReqRequesterDeviceID},
		},
		Data: GetSenderKeyDistributionStatusInput{RoomID: 4},
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.False(t, out.OwnDeviceSenderKeyExists)
	require.NotNil(t, out.RequestableSources)
	require.NotNil(t, out.AvailableFromSources)
	require.NotNil(t, out.AvailableToTargets)
	require.NotNil(t, out.PendingReceivers)
	require.NotNil(t, out.PendingFromSources)
	assert.Empty(t, out.PendingReceivers)
	assert.Empty(t, out.AvailableFromSources)
	assert.Empty(t, out.AvailableToTargets)
	assert.Empty(t, out.RequestableSources)
	assert.Empty(t, out.PendingFromSources)
}

func TestGetSenderKeyDistributionStatusUseCase_PendingFromMembers(t *testing.T) {
	callerMemberID := chatmember.ID(10)
	otherMemberID := chatmember.ID(11)

	uc := NewGetSenderKeyDistributionStatusUseCase(
		&senderStatusParticipantRepoStub{
			findByUserID: func(context.Context, sharedDomain.UserID) (*participant.Participant, error) {
				return &participant.Participant{ID: participant.ID(1)}, nil
			},
		},
		&senderStatusChatMemberRepoStub{
			findByRoomAndParticipant: func(context.Context, chatroom.ID, participant.ID) (*chatmember.ChatMember, error) {
				return &chatmember.ChatMember{ID: callerMemberID}, nil
			},
			findByRoom: func(context.Context, chatroom.ID) ([]*chatmember.ChatMember, error) {
				return []*chatmember.ChatMember{
					{ID: callerMemberID},
					{ID: otherMemberID},
				}, nil
			},
		},
		&senderStatusAccountRepoStub{},
		&senderStatusUserRepoStub{},
		&senderStatusMemberSenderKeyRepoStub{
			findLatestForMember: func(_ context.Context, chatMemberID chatmember.ID) ([]*membersenderkey.MemberSenderKey, error) {
				if chatMemberID == otherMemberID {
					return []*membersenderkey.MemberSenderKey{{
						ChatMemberID:     otherMemberID,
						SenderDeviceID:   senderKeyReqProviderDeviceID,
						SenderKeyVersion: 7,
						ChainID:          membersenderkey.ChainID(7),
					}}, nil
				}
				return nil, membersenderkey.ErrNotFound
			},
		},
		&senderStatusDistributionRepoStub{
			findLatest: func(_ context.Context, senderMemberID chatmember.ID, _ sharedDomain.DeviceID, receiverMemberID chatmember.ID, _ sharedDomain.DeviceID) (*senderkeydistribution.SenderKeyDistribution, error) {
				if senderMemberID == otherMemberID && receiverMemberID == callerMemberID {
					return nil, senderkeydistribution.ErrNotFound
				}
				return nil, senderkeydistribution.ErrNotFound
			},
		},
		&senderStatusReceiptRepoStub{},
	)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[GetSenderKeyDistributionStatusInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{UserID: sharedDomain.UserID(100)},
			Request: appShared.RequestContext{DeviceID: senderKeyReqRequesterDeviceID},
		},
		Data: GetSenderKeyDistributionStatusInput{RoomID: 4},
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.False(t, out.OwnDeviceSenderKeyExists)
	assert.Equal(t, []SenderKeyDeviceRef{{MemberID: int64(otherMemberID), DeviceID: senderKeyReqProviderDeviceID.String()}}, out.RequestableSources)
	assert.Empty(t, out.AvailableFromSources)
	assert.Empty(t, out.AvailableToTargets)
	assert.Equal(t, []SenderKeyDeviceRef{{MemberID: int64(otherMemberID), DeviceID: senderKeyReqProviderDeviceID.String()}}, out.PendingFromSources)
}

func TestGetSenderKeyDistributionStatusUseCase_OwnSenderKeyExists(t *testing.T) {
	callerMemberID := chatmember.ID(10)
	otherMemberID := chatmember.ID(11)

	uc := NewGetSenderKeyDistributionStatusUseCase(
		&senderStatusParticipantRepoStub{
			findByUserID: func(context.Context, sharedDomain.UserID) (*participant.Participant, error) {
				return &participant.Participant{ID: participant.ID(1)}, nil
			},
		},
		&senderStatusChatMemberRepoStub{
			findByRoomAndParticipant: func(context.Context, chatroom.ID, participant.ID) (*chatmember.ChatMember, error) {
				return &chatmember.ChatMember{ID: callerMemberID}, nil
			},
			findByRoom: func(context.Context, chatroom.ID) ([]*chatmember.ChatMember, error) {
				return []*chatmember.ChatMember{
					{ID: callerMemberID},
					{ID: otherMemberID},
				}, nil
			},
		},
		&senderStatusAccountRepoStub{},
		&senderStatusUserRepoStub{},
		&senderStatusMemberSenderKeyRepoStub{
			findLatestForMember: func(_ context.Context, chatMemberID chatmember.ID) ([]*membersenderkey.MemberSenderKey, error) {
				if chatMemberID == callerMemberID {
					return []*membersenderkey.MemberSenderKey{{
						ChatMemberID:     callerMemberID,
						SenderDeviceID:   senderKeyReqRequesterDeviceID,
						SenderKeyVersion: 7,
						ChainID:          membersenderkey.ChainID(7),
					}}, nil
				}
				if chatMemberID == otherMemberID {
					return []*membersenderkey.MemberSenderKey{{
						ChatMemberID:     otherMemberID,
						SenderDeviceID:   senderKeyReqProviderDeviceID,
						SenderKeyVersion: 5,
						ChainID:          membersenderkey.ChainID(5),
					}}, nil
				}
				return nil, membersenderkey.ErrNotFound
			},
		},
		&senderStatusDistributionRepoStub{
			findLatest: func(_ context.Context, senderMemberID chatmember.ID, senderDeviceID sharedDomain.DeviceID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) (*senderkeydistribution.SenderKeyDistribution, error) {
				switch {
				case senderMemberID == otherMemberID && senderDeviceID == senderKeyReqProviderDeviceID && receiverMemberID == callerMemberID && receiverDeviceID == senderKeyReqRequesterDeviceID:
					return &senderkeydistribution.SenderKeyDistribution{
						SenderMemberID:   otherMemberID,
						ReceiverMemberID: callerMemberID,
						SenderKeyVersion: 5,
						Status:           senderkeydistribution.StatusAvailable,
					}, nil
				case senderMemberID == callerMemberID && senderDeviceID == senderKeyReqRequesterDeviceID && receiverMemberID == otherMemberID && receiverDeviceID == senderKeyReqRequesterDeviceID:
					return &senderkeydistribution.SenderKeyDistribution{
						SenderMemberID:   callerMemberID,
						ReceiverMemberID: otherMemberID,
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

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[GetSenderKeyDistributionStatusInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{UserID: sharedDomain.UserID(100)},
			Request: appShared.RequestContext{DeviceID: senderKeyReqRequesterDeviceID},
		},
		Data: GetSenderKeyDistributionStatusInput{RoomID: 4},
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, out.OwnDeviceSenderKeyExists)
	assert.Empty(t, out.RequestableSources)
	assert.Equal(t, []SenderKeyDeviceRef{{MemberID: int64(otherMemberID), DeviceID: senderKeyReqProviderDeviceID.String()}}, out.AvailableFromSources)
	assert.Empty(t, out.PendingReceivers)
	assert.Empty(t, out.AvailableToTargets)
	assert.Empty(t, out.PendingFromSources)
}

func TestGetSenderKeyDistributionStatusUseCase_PendingReceiversRequiresFreshUpload(t *testing.T) {
	callerMemberID := chatmember.ID(10)
	otherMemberID := chatmember.ID(11)

	uc := NewGetSenderKeyDistributionStatusUseCase(
		&senderStatusParticipantRepoStub{
			findByUserID: func(context.Context, sharedDomain.UserID) (*participant.Participant, error) {
				return &participant.Participant{ID: participant.ID(1)}, nil
			},
		},
		&senderStatusChatMemberRepoStub{
			findByRoomAndParticipant: func(context.Context, chatroom.ID, participant.ID) (*chatmember.ChatMember, error) {
				return &chatmember.ChatMember{ID: callerMemberID}, nil
			},
			findByRoom: func(context.Context, chatroom.ID) ([]*chatmember.ChatMember, error) {
				return []*chatmember.ChatMember{
					{ID: callerMemberID},
					{ID: otherMemberID},
				}, nil
			},
		},
		&senderStatusAccountRepoStub{},
		&senderStatusUserRepoStub{},
		&senderStatusMemberSenderKeyRepoStub{
			findLatestForMember: func(_ context.Context, chatMemberID chatmember.ID) ([]*membersenderkey.MemberSenderKey, error) {
				switch chatMemberID {
				case callerMemberID:
					return []*membersenderkey.MemberSenderKey{{
						ChatMemberID:     callerMemberID,
						SenderDeviceID:   senderKeyReqRequesterDeviceID,
						SenderKeyVersion: 9,
						ChainID:          membersenderkey.ChainID(9),
					}}, nil
				case otherMemberID:
					return []*membersenderkey.MemberSenderKey{{
						ChatMemberID:     otherMemberID,
						SenderDeviceID:   senderKeyReqProviderDeviceID,
						SenderKeyVersion: 5,
						ChainID:          membersenderkey.ChainID(5),
					}}, nil
				default:
					return nil, membersenderkey.ErrNotFound
				}
			},
		},
		&senderStatusDistributionRepoStub{
			findLatest: func(_ context.Context, senderMemberID chatmember.ID, senderDeviceID sharedDomain.DeviceID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) (*senderkeydistribution.SenderKeyDistribution, error) {
				switch {
				case senderMemberID == otherMemberID && senderDeviceID == senderKeyReqProviderDeviceID && receiverMemberID == callerMemberID && receiverDeviceID == senderKeyReqRequesterDeviceID:
					return &senderkeydistribution.SenderKeyDistribution{
						SenderMemberID:   otherMemberID,
						ReceiverMemberID: callerMemberID,
						SenderKeyVersion: 5,
						Status:           senderkeydistribution.StatusConsumed,
					}, nil
				case senderMemberID == callerMemberID && senderDeviceID == senderKeyReqRequesterDeviceID && receiverMemberID == otherMemberID && receiverDeviceID == senderKeyReqRequesterDeviceID:
					return &senderkeydistribution.SenderKeyDistribution{
						SenderMemberID:   callerMemberID,
						ReceiverMemberID: otherMemberID,
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

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[GetSenderKeyDistributionStatusInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{UserID: sharedDomain.UserID(100)},
			Request: appShared.RequestContext{DeviceID: senderKeyReqRequesterDeviceID},
		},
		Data: GetSenderKeyDistributionStatusInput{RoomID: 4},
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, out.OwnDeviceSenderKeyExists)
	assert.Empty(t, out.AvailableToTargets)
	assert.Equal(t, []SenderKeyDeviceRef{{MemberID: int64(otherMemberID), DeviceID: senderKeyReqRequesterDeviceID.String()}}, out.PendingReceivers)
}

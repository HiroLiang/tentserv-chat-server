package usecase

import (
	"context"
	"testing"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	sharedDomain "github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type senderStatusParticipantRepoStub struct {
	findByUserID func(ctx context.Context, userID sharedDomain.UserID) (*participant.Participant, error)
}

func (s *senderStatusParticipantRepoStub) FindByID(context.Context, participant.ID) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
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
	findLatest func(ctx context.Context, chatMemberID chatmember.ID) (*membersenderkey.MemberSenderKey, error)
}

func (s *senderStatusMemberSenderKeyRepoStub) FindLatest(ctx context.Context, chatMemberID chatmember.ID) (*membersenderkey.MemberSenderKey, error) {
	if s.findLatest != nil {
		return s.findLatest(ctx, chatMemberID)
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
	findLatest                     func(ctx context.Context, senderMemberID, receiverMemberID chatmember.ID) (*senderkeydistribution.SenderKeyDistribution, error)
	findAvailableByRoomAndReceiver func(ctx context.Context, roomID chatroom.ID, receiverMemberID chatmember.ID) ([]*senderkeydistribution.SenderKeyDistribution, error)
}

func (s *senderStatusDistributionRepoStub) UpsertBatch(context.Context, []*senderkeydistribution.SenderKeyDistribution) error {
	return nil
}

func (s *senderStatusDistributionRepoStub) FindPendingReceivers(ctx context.Context, senderMemberID chatmember.ID, latestChainID int64) ([]chatmember.ID, error) {
	return []chatmember.ID{}, nil
}

func (s *senderStatusDistributionRepoStub) UpsertAvailable(context.Context, *senderkeydistribution.SenderKeyDistribution) error {
	return nil
}

func (s *senderStatusDistributionRepoStub) FindLatest(ctx context.Context, senderMemberID, receiverMemberID chatmember.ID) (*senderkeydistribution.SenderKeyDistribution, error) {
	if s.findLatest != nil {
		return s.findLatest(ctx, senderMemberID, receiverMemberID)
	}
	return nil, senderkeydistribution.ErrNotFound
}

func (s *senderStatusDistributionRepoStub) FindAvailableByRoomAndReceiver(ctx context.Context, roomID chatroom.ID, receiverMemberID chatmember.ID) ([]*senderkeydistribution.SenderKeyDistribution, error) {
	if s.findAvailableByRoomAndReceiver != nil {
		return s.findAvailableByRoomAndReceiver(ctx, roomID, receiverMemberID)
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
		&senderStatusMemberSenderKeyRepoStub{
			findLatest: func(_ context.Context, chatMemberID chatmember.ID) (*membersenderkey.MemberSenderKey, error) {
				return nil, membersenderkey.ErrNotFound
			},
		},
		&senderStatusDistributionRepoStub{},
	)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[GetSenderKeyDistributionStatusInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{UserID: sharedDomain.UserID(100)},
		},
		Data: GetSenderKeyDistributionStatusInput{RoomID: 4},
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.False(t, out.OwnSenderKeyExists)
	require.NotNil(t, out.RequestableMemberIDs)
	require.NotNil(t, out.AvailableFromMemberIDs)
	require.NotNil(t, out.AvailableToMemberIDs)
	require.NotNil(t, out.PendingReceivers)
	require.NotNil(t, out.PendingFromMembers)
	assert.Empty(t, out.PendingReceivers)
	assert.Empty(t, out.AvailableFromMemberIDs)
	assert.Empty(t, out.AvailableToMemberIDs)
	assert.Equal(t, []int64{11}, out.RequestableMemberIDs)
	assert.Equal(t, []int64{11}, out.PendingFromMembers)
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
		&senderStatusMemberSenderKeyRepoStub{
			findLatest: func(_ context.Context, chatMemberID chatmember.ID) (*membersenderkey.MemberSenderKey, error) {
				if chatMemberID == otherMemberID {
					return &membersenderkey.MemberSenderKey{
						ChatMemberID:     otherMemberID,
						SenderKeyVersion: 7,
						ChainID:          membersenderkey.ChainID(7),
					}, nil
				}
				return nil, membersenderkey.ErrNotFound
			},
		},
		&senderStatusDistributionRepoStub{
			findLatest: func(_ context.Context, senderMemberID, receiverMemberID chatmember.ID) (*senderkeydistribution.SenderKeyDistribution, error) {
				if senderMemberID == otherMemberID && receiverMemberID == callerMemberID {
					return nil, senderkeydistribution.ErrNotFound
				}
				return nil, senderkeydistribution.ErrNotFound
			},
		},
	)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[GetSenderKeyDistributionStatusInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{UserID: sharedDomain.UserID(100)},
		},
		Data: GetSenderKeyDistributionStatusInput{RoomID: 4},
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.False(t, out.OwnSenderKeyExists)
	assert.Equal(t, []int64{int64(otherMemberID)}, out.RequestableMemberIDs)
	assert.Empty(t, out.AvailableFromMemberIDs)
	assert.Empty(t, out.AvailableToMemberIDs)
	assert.Equal(t, []int64{int64(otherMemberID)}, out.PendingFromMembers)
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
		&senderStatusMemberSenderKeyRepoStub{
			findLatest: func(_ context.Context, chatMemberID chatmember.ID) (*membersenderkey.MemberSenderKey, error) {
				if chatMemberID == callerMemberID {
					return &membersenderkey.MemberSenderKey{
						ChatMemberID:     callerMemberID,
						SenderKeyVersion: 7,
						ChainID:          membersenderkey.ChainID(7),
					}, nil
				}
				if chatMemberID == otherMemberID {
					return &membersenderkey.MemberSenderKey{
						ChatMemberID:     otherMemberID,
						SenderKeyVersion: 5,
						ChainID:          membersenderkey.ChainID(5),
					}, nil
				}
				return nil, membersenderkey.ErrNotFound
			},
		},
		&senderStatusDistributionRepoStub{
			findLatest: func(_ context.Context, senderMemberID, receiverMemberID chatmember.ID) (*senderkeydistribution.SenderKeyDistribution, error) {
				switch {
				case senderMemberID == otherMemberID && receiverMemberID == callerMemberID:
					return &senderkeydistribution.SenderKeyDistribution{
						SenderMemberID:   otherMemberID,
						ReceiverMemberID: callerMemberID,
						SenderKeyVersion: 5,
						Status:           senderkeydistribution.StatusAvailable,
					}, nil
				case senderMemberID == callerMemberID && receiverMemberID == otherMemberID:
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
	)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[GetSenderKeyDistributionStatusInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{UserID: sharedDomain.UserID(100)},
		},
		Data: GetSenderKeyDistributionStatusInput{RoomID: 4},
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, out.OwnSenderKeyExists)
	assert.Empty(t, out.RequestableMemberIDs)
	assert.Equal(t, []int64{int64(otherMemberID)}, out.AvailableFromMemberIDs)
	assert.Empty(t, out.PendingReceivers)
	assert.Empty(t, out.AvailableToMemberIDs)
	assert.Empty(t, out.PendingFromMembers)
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
		&senderStatusMemberSenderKeyRepoStub{
			findLatest: func(_ context.Context, chatMemberID chatmember.ID) (*membersenderkey.MemberSenderKey, error) {
				switch chatMemberID {
				case callerMemberID:
					return &membersenderkey.MemberSenderKey{
						ChatMemberID:     callerMemberID,
						SenderKeyVersion: 9,
						ChainID:          membersenderkey.ChainID(9),
					}, nil
				case otherMemberID:
					return &membersenderkey.MemberSenderKey{
						ChatMemberID:     otherMemberID,
						SenderKeyVersion: 5,
						ChainID:          membersenderkey.ChainID(5),
					}, nil
				default:
					return nil, membersenderkey.ErrNotFound
				}
			},
		},
		&senderStatusDistributionRepoStub{
			findLatest: func(_ context.Context, senderMemberID, receiverMemberID chatmember.ID) (*senderkeydistribution.SenderKeyDistribution, error) {
				switch {
				case senderMemberID == otherMemberID && receiverMemberID == callerMemberID:
					return &senderkeydistribution.SenderKeyDistribution{
						SenderMemberID:   otherMemberID,
						ReceiverMemberID: callerMemberID,
						SenderKeyVersion: 5,
						Status:           senderkeydistribution.StatusConsumed,
					}, nil
				case senderMemberID == callerMemberID && receiverMemberID == otherMemberID:
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
	)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[GetSenderKeyDistributionStatusInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{UserID: sharedDomain.UserID(100)},
		},
		Data: GetSenderKeyDistributionStatusInput{RoomID: 4},
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, out.OwnSenderKeyExists)
	assert.Empty(t, out.AvailableToMemberIDs)
	assert.Equal(t, []int64{int64(otherMemberID)}, out.PendingReceivers)
}

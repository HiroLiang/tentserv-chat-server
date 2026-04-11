package usecase

import (
	"context"
	"testing"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockedMessagesFriendshipRepoStub struct {
	rows []*friendship.Friendship
}

func (s *blockedMessagesFriendshipRepoStub) FindBetweenUsers(context.Context, shared.UserID, shared.UserID) ([]*friendship.Friendship, error) {
	return s.rows, nil
}

func (s *blockedMessagesFriendshipRepoStub) FindByUserID(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}

func (s *blockedMessagesFriendshipRepoStub) FindAllByUserID(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}

func (s *blockedMessagesFriendshipRepoStub) FindPendingByUserID(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}

func (s *blockedMessagesFriendshipRepoStub) Create(context.Context, shared.UserID, shared.UserID) error {
	return nil
}

func (s *blockedMessagesFriendshipRepoStub) CreateBlocked(context.Context, shared.UserID, shared.UserID) error {
	return nil
}

func (s *blockedMessagesFriendshipRepoStub) FindByID(context.Context, int64) (*friendship.Friendship, error) {
	return nil, friendship.ErrFriendshipNotFound
}

func (s *blockedMessagesFriendshipRepoStub) FindByUserIDAndFriendID(context.Context, shared.UserID, shared.UserID) (*friendship.Friendship, error) {
	return nil, friendship.ErrFriendshipNotFound
}

func (s *blockedMessagesFriendshipRepoStub) FindPendingByFriendID(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}

func (s *blockedMessagesFriendshipRepoStub) UpdateStatus(context.Context, int64, friendship.Status) error {
	return nil
}

func (s *blockedMessagesFriendshipRepoStub) Delete(context.Context, int64) error {
	return nil
}

func TestBlockedMembersForCaller_SplitsPeerAndCallerBlockFlags(t *testing.T) {
	callerUserID := shared.UserID(10)
	peerUserID := shared.UserID(20)
	callerParticipantID := participant.ID(100)
	peerParticipantID := participant.ID(200)
	peerMemberID := chatmember.ID(300)

	participantRepo := &getRoomsParticipantRepoStub{
		byID: map[participant.ID]*participant.Participant{
			callerParticipantID: {ID: callerParticipantID, Type: participant.UserType, UserID: &callerUserID},
			peerParticipantID:   {ID: peerParticipantID, Type: participant.UserType, UserID: &peerUserID},
		},
	}
	members := []*chatmember.ChatMember{
		{ID: 1, ParticipantID: callerParticipantID},
		{ID: peerMemberID, ParticipantID: peerParticipantID},
	}

	t.Run("peer blocked caller", func(t *testing.T) {
		friendshipRepo := &blockedMessagesFriendshipRepoStub{
			rows: []*friendship.Friendship{{
				UserID:   peerUserID,
				FriendID: callerUserID,
				Status:   friendship.StatusBlocked,
			}},
		}

		blocked, blockedByPeer, blockedByMe, err := blockedMembersForCaller(
			context.Background(),
			friendshipRepo,
			participantRepo,
			callerUserID,
			callerParticipantID,
			members,
		)

		require.NoError(t, err)
		assert.Contains(t, blocked, peerMemberID)
		assert.True(t, blockedByPeer)
		assert.False(t, blockedByMe)
	})

	t.Run("caller blocked peer", func(t *testing.T) {
		friendshipRepo := &blockedMessagesFriendshipRepoStub{
			rows: []*friendship.Friendship{{
				UserID:   callerUserID,
				FriendID: peerUserID,
				Status:   friendship.StatusBlocked,
			}},
		}

		blocked, blockedByPeer, blockedByMe, err := blockedMembersForCaller(
			context.Background(),
			friendshipRepo,
			participantRepo,
			callerUserID,
			callerParticipantID,
			members,
		)

		require.NoError(t, err)
		assert.Contains(t, blocked, peerMemberID)
		assert.False(t, blockedByPeer)
		assert.True(t, blockedByMe)
	})
}

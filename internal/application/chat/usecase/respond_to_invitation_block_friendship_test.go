package usecase

import (
	"context"
	"fmt"
	"testing"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatinvitation"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type respondBlockTxStub struct {
	committed bool
}

func (t *respondBlockTxStub) Commit() error {
	t.committed = true
	return nil
}

func (t *respondBlockTxStub) Rollback() error {
	return nil
}

type respondBlockUOWStub struct {
	tx *respondBlockTxStub
}

func (u *respondBlockUOWStub) Begin(ctx context.Context) (context.Context, transaction.Transaction, error) {
	if u.tx == nil {
		u.tx = &respondBlockTxStub{}
	}
	return ctx, u.tx, nil
}

type respondBlockParticipantRepoStub struct {
	byUserID map[shared.UserID]*participant.Participant
	byID     map[participant.ID]*participant.Participant
}

func (s *respondBlockParticipantRepoStub) FindByID(ctx context.Context, id participant.ID) (*participant.Participant, error) {
	p, ok := s.byID[id]
	if !ok {
		return nil, participant.ErrNotFound
	}
	return p, nil
}

func (s *respondBlockParticipantRepoStub) FindByUserID(ctx context.Context, userID shared.UserID) (*participant.Participant, error) {
	p, ok := s.byUserID[userID]
	if !ok {
		return nil, participant.ErrNotFound
	}
	return p, nil
}

func (s *respondBlockParticipantRepoStub) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *respondBlockParticipantRepoStub) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *respondBlockParticipantRepoStub) Create(context.Context, *participant.Participant) error {
	return nil
}

type respondBlockChatMemberRepoStub struct{}

func (s *respondBlockChatMemberRepoStub) FindByID(context.Context, chatmember.ID) (*chatmember.ChatMember, error) {
	return nil, chatmember.ErrNotFound
}

func (s *respondBlockChatMemberRepoStub) FindByRoomAndParticipant(context.Context, chatroom.ID, participant.ID) (*chatmember.ChatMember, error) {
	return nil, chatmember.ErrNotFound
}

func (s *respondBlockChatMemberRepoStub) FindByRoom(context.Context, chatroom.ID) ([]*chatmember.ChatMember, error) {
	return nil, nil
}

func (s *respondBlockChatMemberRepoStub) FindByParticipant(context.Context, participant.ID) ([]*chatmember.ChatMember, error) {
	return nil, nil
}

func (s *respondBlockChatMemberRepoStub) Add(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (s *respondBlockChatMemberRepoStub) Update(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (s *respondBlockChatMemberRepoStub) SoftDelete(context.Context, chatmember.ID) error {
	return nil
}

func (s *respondBlockChatMemberRepoStub) Remove(context.Context, chatroom.ID, participant.ID) error {
	return nil
}

type respondBlockInvitationRepoStub struct {
	invitation         *chatinvitation.ChatInvitation
	updateStatusCalls  []chatinvitation.Status
	updateStatusCalled int
}

func (s *respondBlockInvitationRepoStub) Create(context.Context, *chatinvitation.ChatInvitation) error {
	return nil
}

func (s *respondBlockInvitationRepoStub) FindByID(context.Context, chatinvitation.ID) (*chatinvitation.ChatInvitation, error) {
	if s.invitation == nil {
		return nil, chatinvitation.ErrNotFound
	}
	return s.invitation, nil
}

func (s *respondBlockInvitationRepoStub) FindByRoomAndInvitee(context.Context, chatroom.ID, participant.ID) (*chatinvitation.ChatInvitation, error) {
	return nil, chatinvitation.ErrNotFound
}

func (s *respondBlockInvitationRepoStub) FindPendingByRoomAndInviter(context.Context, chatroom.ID, participant.ID) (*chatinvitation.ChatInvitation, error) {
	return nil, chatinvitation.ErrNotFound
}

func (s *respondBlockInvitationRepoStub) FindByRoom(context.Context, chatroom.ID) ([]*chatinvitation.ChatInvitation, error) {
	return nil, nil
}

func (s *respondBlockInvitationRepoStub) UpdateStatus(ctx context.Context, id chatinvitation.ID, status chatinvitation.Status) error {
	s.updateStatusCalled++
	s.updateStatusCalls = append(s.updateStatusCalls, status)
	return nil
}

type respondBlockFriendshipRepoStub struct {
	records map[[2]shared.UserID]*friendship.Friendship

	createdPairs [][2]shared.UserID
	updatedIDs   []int64
	deletedIDs   []int64

	updateStatusErrByID map[int64]error
	createBlockedErr    error
	deleteErrByID       map[int64]error
}

func (s *respondBlockFriendshipRepoStub) FindByUserID(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}

func (s *respondBlockFriendshipRepoStub) FindAllByUserID(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}

func (s *respondBlockFriendshipRepoStub) FindPendingByUserID(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}

func (s *respondBlockFriendshipRepoStub) Create(context.Context, shared.UserID, shared.UserID) error {
	return nil
}

func (s *respondBlockFriendshipRepoStub) CreateBlocked(ctx context.Context, userID, friendID shared.UserID) error {
	if s.createBlockedErr != nil {
		return s.createBlockedErr
	}
	key := [2]shared.UserID{userID, friendID}
	if _, exists := s.records[key]; exists {
		return fmt.Errorf("duplicate key value violates unique constraint")
	}
	s.createdPairs = append(s.createdPairs, key)
	s.records[key] = &friendship.Friendship{
		ID:       int64(len(s.records) + 100),
		UserID:   userID,
		FriendID: friendID,
		Status:   friendship.StatusBlocked,
	}
	return nil
}

func (s *respondBlockFriendshipRepoStub) FindByID(context.Context, int64) (*friendship.Friendship, error) {
	return nil, friendship.ErrFriendshipNotFound
}

func (s *respondBlockFriendshipRepoStub) FindByUserIDAndFriendID(ctx context.Context, userID, friendID shared.UserID) (*friendship.Friendship, error) {
	key := [2]shared.UserID{userID, friendID}
	fs, ok := s.records[key]
	if !ok {
		return nil, friendship.ErrFriendshipNotFound
	}
	return fs, nil
}

func (s *respondBlockFriendshipRepoStub) FindBetweenUsers(ctx context.Context, userID1, userID2 shared.UserID) ([]*friendship.Friendship, error) {
	out := make([]*friendship.Friendship, 0, 2)
	if fs, ok := s.records[[2]shared.UserID{userID1, userID2}]; ok {
		out = append(out, fs)
	}
	if fs, ok := s.records[[2]shared.UserID{userID2, userID1}]; ok {
		out = append(out, fs)
	}
	if len(out) == 0 {
		return nil, friendship.ErrFriendshipNotFound
	}
	return out, nil
}

func (s *respondBlockFriendshipRepoStub) FindPendingByFriendID(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}

func (s *respondBlockFriendshipRepoStub) UpdateStatus(ctx context.Context, id int64, status friendship.Status) error {
	if err, ok := s.updateStatusErrByID[id]; ok {
		return err
	}
	s.updatedIDs = append(s.updatedIDs, id)
	for _, fs := range s.records {
		if fs.ID == id {
			fs.Status = status
			return nil
		}
	}
	return nil
}

func (s *respondBlockFriendshipRepoStub) Delete(ctx context.Context, id int64) error {
	if err, ok := s.deleteErrByID[id]; ok {
		return err
	}
	s.deletedIDs = append(s.deletedIDs, id)
	for key, fs := range s.records {
		if fs.ID == id {
			delete(s.records, key)
			break
		}
	}
	return nil
}

func TestRespondToInvitationUseCase_BlockCreatesForwardAndDeletesReverse(t *testing.T) {
	callerUserID := shared.UserID(100)
	inviterUserID := shared.UserID(200)
	callerParticipantID := participant.ID(10)
	inviterParticipantID := participant.ID(20)

	uow := &respondBlockUOWStub{}
	invitationRepo := &respondBlockInvitationRepoStub{
		invitation: &chatinvitation.ChatInvitation{
			ID:             chatinvitation.ID(1),
			RoomID:         chatroom.ID(7),
			InviterID:      inviterParticipantID,
			InviteeID:      callerParticipantID,
			Status:         chatinvitation.Pending,
			InvitationType: chatinvitation.Invitation,
		},
	}
	participantRepo := &respondBlockParticipantRepoStub{
		byUserID: map[shared.UserID]*participant.Participant{
			callerUserID: {ID: callerParticipantID, UserID: &callerUserID},
		},
		byID: map[participant.ID]*participant.Participant{
			inviterParticipantID: {ID: inviterParticipantID, UserID: &inviterUserID},
		},
	}
	friendshipRepo := &respondBlockFriendshipRepoStub{
		records: map[[2]shared.UserID]*friendship.Friendship{
			[2]shared.UserID{inviterUserID, callerUserID}: {ID: 33, UserID: inviterUserID, FriendID: callerUserID, Status: friendship.StatusAccepted},
		},
	}

	uc := NewRespondToInvitationUseCase(uow, participantRepo, &respondBlockChatMemberRepoStub{}, invitationRepo, friendshipRepo, nil)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[RespondToInvitationInput]{
		Base: appShared.BaseContext{Auth: &appShared.AuthContext{UserID: callerUserID}},
		Data: RespondToInvitationInput{InvitationID: 1, Action: "block"},
	})

	require.NoError(t, err)
	assert.Equal(t, string(chatinvitation.Blocked), out.Status)
	assert.Equal(t, [][2]shared.UserID{{callerUserID, inviterUserID}}, friendshipRepo.createdPairs)
	assert.Equal(t, []int64{33}, friendshipRepo.deletedIDs)
	require.NotNil(t, uow.tx)
	assert.True(t, uow.tx.committed)
}

func TestRespondToInvitationUseCase_BlockKeepsOnlyForwardBlockedWhenBothDirectionsExist(t *testing.T) {
	callerUserID := shared.UserID(100)
	inviterUserID := shared.UserID(200)
	callerParticipantID := participant.ID(10)
	inviterParticipantID := participant.ID(20)

	uow := &respondBlockUOWStub{}
	invitationRepo := &respondBlockInvitationRepoStub{
		invitation: &chatinvitation.ChatInvitation{
			ID:             chatinvitation.ID(1),
			RoomID:         chatroom.ID(7),
			InviterID:      inviterParticipantID,
			InviteeID:      callerParticipantID,
			Status:         chatinvitation.Pending,
			InvitationType: chatinvitation.Invitation,
		},
	}
	participantRepo := &respondBlockParticipantRepoStub{
		byUserID: map[shared.UserID]*participant.Participant{
			callerUserID: {ID: callerParticipantID, UserID: &callerUserID},
		},
		byID: map[participant.ID]*participant.Participant{
			inviterParticipantID: {ID: inviterParticipantID, UserID: &inviterUserID},
		},
	}
	friendshipRepo := &respondBlockFriendshipRepoStub{
		records: map[[2]shared.UserID]*friendship.Friendship{
			[2]shared.UserID{callerUserID, inviterUserID}: {ID: 11, UserID: callerUserID, FriendID: inviterUserID, Status: friendship.StatusAccepted},
			[2]shared.UserID{inviterUserID, callerUserID}: {ID: 12, UserID: inviterUserID, FriendID: callerUserID, Status: friendship.StatusAccepted},
		},
	}

	uc := NewRespondToInvitationUseCase(uow, participantRepo, &respondBlockChatMemberRepoStub{}, invitationRepo, friendshipRepo, nil)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[RespondToInvitationInput]{
		Base: appShared.BaseContext{Auth: &appShared.AuthContext{UserID: callerUserID}},
		Data: RespondToInvitationInput{InvitationID: 1, Action: "block"},
	})

	require.NoError(t, err)
	assert.Equal(t, string(chatinvitation.Blocked), out.Status)
	assert.Empty(t, friendshipRepo.createdPairs)
	assert.Equal(t, []int64{11}, friendshipRepo.updatedIDs)
	assert.Equal(t, []int64{12}, friendshipRepo.deletedIDs)

	forward, ok := friendshipRepo.records[[2]shared.UserID{callerUserID, inviterUserID}]
	require.True(t, ok)
	assert.Equal(t, friendship.StatusBlocked, forward.Status)

	_, reverseExists := friendshipRepo.records[[2]shared.UserID{inviterUserID, callerUserID}]
	assert.False(t, reverseExists)
}

func TestRespondToInvitationUseCase_BlockReturnsErrorWhenFriendshipSyncFails(t *testing.T) {
	callerUserID := shared.UserID(100)
	inviterUserID := shared.UserID(200)
	callerParticipantID := participant.ID(10)
	inviterParticipantID := participant.ID(20)

	uow := &respondBlockUOWStub{}
	invitationRepo := &respondBlockInvitationRepoStub{
		invitation: &chatinvitation.ChatInvitation{
			ID:             chatinvitation.ID(1),
			RoomID:         chatroom.ID(7),
			InviterID:      inviterParticipantID,
			InviteeID:      callerParticipantID,
			Status:         chatinvitation.Pending,
			InvitationType: chatinvitation.Invitation,
		},
	}
	participantRepo := &respondBlockParticipantRepoStub{
		byUserID: map[shared.UserID]*participant.Participant{
			callerUserID: {ID: callerParticipantID, UserID: &callerUserID},
		},
		byID: map[participant.ID]*participant.Participant{
			inviterParticipantID: {ID: inviterParticipantID, UserID: &inviterUserID},
		},
	}
	friendshipRepo := &respondBlockFriendshipRepoStub{
		records: map[[2]shared.UserID]*friendship.Friendship{
			[2]shared.UserID{callerUserID, inviterUserID}: {ID: 21, UserID: callerUserID, FriendID: inviterUserID, Status: friendship.StatusAccepted},
		},
		updateStatusErrByID: map[int64]error{
			21: assert.AnError,
		},
	}

	uc := NewRespondToInvitationUseCase(uow, participantRepo, &respondBlockChatMemberRepoStub{}, invitationRepo, friendshipRepo, nil)

	_, err := uc.Execute(context.Background(), appShared.UseCaseInput[RespondToInvitationInput]{
		Base: appShared.BaseContext{Auth: &appShared.AuthContext{UserID: callerUserID}},
		Data: RespondToInvitationInput{InvitationID: 1, Action: "block"},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvitationCreate)
	assert.Equal(t, 0, invitationRepo.updateStatusCalled)
	require.NotNil(t, uow.tx)
	assert.False(t, uow.tx.committed)
}

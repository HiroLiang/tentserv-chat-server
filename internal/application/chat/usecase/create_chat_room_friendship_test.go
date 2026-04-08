package usecase

import (
	"context"
	"testing"
	"time"

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

type createRoomTxStub struct {
	committed bool
}

func (t *createRoomTxStub) Commit() error {
	t.committed = true
	return nil
}

func (t *createRoomTxStub) Rollback() error {
	return nil
}

type createRoomUOWStub struct {
	tx *createRoomTxStub
}

func (u *createRoomUOWStub) Begin(ctx context.Context) (context.Context, transaction.Transaction, error) {
	if u.tx == nil {
		u.tx = &createRoomTxStub{}
	}
	return ctx, u.tx, nil
}

type createRoomParticipantRepoStub struct {
	byUserID map[shared.UserID]*participant.Participant
}

func (s *createRoomParticipantRepoStub) FindByID(ctx context.Context, id participant.ID) (*participant.Participant, error) {
	for _, p := range s.byUserID {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, participant.ErrNotFound
}

func (s *createRoomParticipantRepoStub) FindByUserID(ctx context.Context, userID shared.UserID) (*participant.Participant, error) {
	p, ok := s.byUserID[userID]
	if !ok {
		return nil, participant.ErrNotFound
	}
	return p, nil
}

func (s *createRoomParticipantRepoStub) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *createRoomParticipantRepoStub) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *createRoomParticipantRepoStub) Create(context.Context, *participant.Participant) error {
	return nil
}

type createRoomChatRoomRepoStub struct {
	findDirectByParticipantsCalled bool
	findDirectByParticipantsRoom   *chatroom.ChatRoom
	createCalls                    int
}

func (s *createRoomChatRoomRepoStub) FindByID(context.Context, chatroom.ID) (*chatroom.ChatRoom, error) {
	return nil, chatroom.ErrNotFound
}

func (s *createRoomChatRoomRepoStub) Create(ctx context.Context, room *chatroom.ChatRoom) error {
	s.createCalls++
	if room.ID == 0 {
		room.ID = chatroom.ID(100 + s.createCalls)
	}
	if room.CreatedAt.IsZero() {
		room.CreatedAt = time.Now()
	}
	return nil
}

func (s *createRoomChatRoomRepoStub) FindDirectByParticipants(context.Context, participant.ID, participant.ID) (*chatroom.ChatRoom, error) {
	s.findDirectByParticipantsCalled = true
	if s.findDirectByParticipantsRoom == nil {
		return nil, chatroom.ErrNotFound
	}
	return s.findDirectByParticipantsRoom, nil
}

func (s *createRoomChatRoomRepoStub) Update(context.Context, *chatroom.ChatRoom) error {
	return nil
}

func (s *createRoomChatRoomRepoStub) SoftDelete(context.Context, chatroom.ID) error {
	return nil
}

type createRoomChatMemberRepoStub struct {
	addCalls int
}

func (s *createRoomChatMemberRepoStub) FindByID(context.Context, chatmember.ID) (*chatmember.ChatMember, error) {
	return nil, chatmember.ErrNotFound
}

func (s *createRoomChatMemberRepoStub) FindByRoomAndParticipant(context.Context, chatroom.ID, participant.ID) (*chatmember.ChatMember, error) {
	return nil, chatmember.ErrNotFound
}

func (s *createRoomChatMemberRepoStub) FindByRoom(context.Context, chatroom.ID) ([]*chatmember.ChatMember, error) {
	return nil, nil
}

func (s *createRoomChatMemberRepoStub) FindByParticipant(context.Context, participant.ID) ([]*chatmember.ChatMember, error) {
	return nil, nil
}

func (s *createRoomChatMemberRepoStub) Add(context.Context, *chatmember.ChatMember) error {
	s.addCalls++
	return nil
}

func (s *createRoomChatMemberRepoStub) Update(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (s *createRoomChatMemberRepoStub) SoftDelete(context.Context, chatmember.ID) error {
	return nil
}

func (s *createRoomChatMemberRepoStub) Remove(context.Context, chatroom.ID, participant.ID) error {
	return nil
}

type createRoomInvitationRepoStub struct {
	createCalls int
}

func (s *createRoomInvitationRepoStub) Create(context.Context, *chatinvitation.ChatInvitation) error {
	s.createCalls++
	return nil
}

func (s *createRoomInvitationRepoStub) FindByID(context.Context, chatinvitation.ID) (*chatinvitation.ChatInvitation, error) {
	return nil, chatinvitation.ErrNotFound
}

func (s *createRoomInvitationRepoStub) FindByRoomAndInvitee(context.Context, chatroom.ID, participant.ID) (*chatinvitation.ChatInvitation, error) {
	return nil, chatinvitation.ErrNotFound
}

func (s *createRoomInvitationRepoStub) FindPendingByRoomAndInviter(context.Context, chatroom.ID, participant.ID) (*chatinvitation.ChatInvitation, error) {
	return nil, chatinvitation.ErrNotFound
}

func (s *createRoomInvitationRepoStub) FindByRoom(context.Context, chatroom.ID) ([]*chatinvitation.ChatInvitation, error) {
	return nil, nil
}

func (s *createRoomInvitationRepoStub) UpdateStatus(context.Context, chatinvitation.ID, chatinvitation.Status) error {
	return nil
}

type createRoomFriendshipRepoStub struct {
	findBetweenUsers func(ctx context.Context, userID1, userID2 shared.UserID) ([]*friendship.Friendship, error)
}

func (s *createRoomFriendshipRepoStub) FindByUserID(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}

func (s *createRoomFriendshipRepoStub) FindAllByUserID(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}

func (s *createRoomFriendshipRepoStub) FindPendingByUserID(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}

func (s *createRoomFriendshipRepoStub) Create(context.Context, shared.UserID, shared.UserID) error {
	return nil
}

func (s *createRoomFriendshipRepoStub) CreateBlocked(context.Context, shared.UserID, shared.UserID) error {
	return nil
}

func (s *createRoomFriendshipRepoStub) FindByID(context.Context, int64) (*friendship.Friendship, error) {
	return nil, friendship.ErrFriendshipNotFound
}

func (s *createRoomFriendshipRepoStub) FindByUserIDAndFriendID(context.Context, shared.UserID, shared.UserID) (*friendship.Friendship, error) {
	return nil, friendship.ErrFriendshipNotFound
}

func (s *createRoomFriendshipRepoStub) FindBetweenUsers(ctx context.Context, userID1, userID2 shared.UserID) ([]*friendship.Friendship, error) {
	if s.findBetweenUsers != nil {
		return s.findBetweenUsers(ctx, userID1, userID2)
	}
	return nil, friendship.ErrFriendshipNotFound
}

func (s *createRoomFriendshipRepoStub) FindPendingByFriendID(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}

func (s *createRoomFriendshipRepoStub) UpdateStatus(context.Context, int64, friendship.Status) error {
	return nil
}

func (s *createRoomFriendshipRepoStub) Delete(context.Context, int64) error {
	return nil
}

func TestCreateChatRoomUseCase_BlocksDirectWhenEitherDirectionBlocked(t *testing.T) {
	currentUserID := shared.UserID(1)
	invitedUserID := shared.UserID(2)

	uow := &createRoomUOWStub{}
	chatRoomRepo := &createRoomChatRoomRepoStub{}
	chatMemberRepo := &createRoomChatMemberRepoStub{}
	invitationRepo := &createRoomInvitationRepoStub{}
	participantRepo := &createRoomParticipantRepoStub{
		byUserID: map[shared.UserID]*participant.Participant{
			currentUserID: {ID: participant.ID(10), UserID: &currentUserID},
			invitedUserID: {ID: participant.ID(20), UserID: &invitedUserID},
		},
	}
	friendshipRepo := &createRoomFriendshipRepoStub{
		findBetweenUsers: func(ctx context.Context, userID1, userID2 shared.UserID) ([]*friendship.Friendship, error) {
			return []*friendship.Friendship{
				{ID: 1, UserID: currentUserID, FriendID: invitedUserID, Status: friendship.StatusAccepted},
				{ID: 2, UserID: invitedUserID, FriendID: currentUserID, Status: friendship.StatusBlocked},
			}, nil
		},
	}

	uc := NewCreateChatRoomUseCase(uow, chatRoomRepo, chatMemberRepo, participantRepo, friendshipRepo, invitationRepo)

	_, err := uc.Execute(context.Background(), appShared.UseCaseInput[CreateChatRoomInput]{
		Base: appShared.BaseContext{Auth: &appShared.AuthContext{UserID: currentUserID}},
		Data: CreateChatRoomInput{
			Name:      "direct room",
			Type:      chatroom.Direct,
			MemberIDs: []int64{int64(invitedUserID)},
		},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUserBlocked)
	assert.False(t, chatRoomRepo.findDirectByParticipantsCalled)
	assert.Equal(t, 0, chatRoomRepo.createCalls)
	assert.Equal(t, 0, invitationRepo.createCalls)
}

func TestCreateChatRoomUseCase_BlocksGroupWhenAnyInviteeBlocked(t *testing.T) {
	currentUserID := shared.UserID(1)
	invitee1 := shared.UserID(2)
	invitee2 := shared.UserID(3)

	uow := &createRoomUOWStub{}
	chatRoomRepo := &createRoomChatRoomRepoStub{}
	chatMemberRepo := &createRoomChatMemberRepoStub{}
	invitationRepo := &createRoomInvitationRepoStub{}
	participantRepo := &createRoomParticipantRepoStub{
		byUserID: map[shared.UserID]*participant.Participant{
			currentUserID: {ID: participant.ID(10), UserID: &currentUserID},
			invitee1:      {ID: participant.ID(20), UserID: &invitee1},
			invitee2:      {ID: participant.ID(30), UserID: &invitee2},
		},
	}
	friendshipRepo := &createRoomFriendshipRepoStub{
		findBetweenUsers: func(ctx context.Context, userID1, userID2 shared.UserID) ([]*friendship.Friendship, error) {
			if userID2 == invitee2 {
				return []*friendship.Friendship{
					{ID: 9, UserID: invitee2, FriendID: currentUserID, Status: friendship.StatusBlocked},
				}, nil
			}
			return nil, friendship.ErrFriendshipNotFound
		},
	}

	uc := NewCreateChatRoomUseCase(uow, chatRoomRepo, chatMemberRepo, participantRepo, friendshipRepo, invitationRepo)

	_, err := uc.Execute(context.Background(), appShared.UseCaseInput[CreateChatRoomInput]{
		Base: appShared.BaseContext{Auth: &appShared.AuthContext{UserID: currentUserID}},
		Data: CreateChatRoomInput{
			Name:      "group room",
			Type:      chatroom.Group,
			MemberIDs: []int64{int64(invitee1), int64(invitee2)},
		},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUserBlocked)
	assert.Equal(t, 0, chatRoomRepo.createCalls)
	assert.Equal(t, 0, invitationRepo.createCalls)
}

func TestCreateChatRoomUseCase_CreatesRoomWhenNoBlockedRelationship(t *testing.T) {
	currentUserID := shared.UserID(1)
	invitee1 := shared.UserID(2)
	invitee2 := shared.UserID(3)

	uow := &createRoomUOWStub{}
	chatRoomRepo := &createRoomChatRoomRepoStub{}
	chatMemberRepo := &createRoomChatMemberRepoStub{}
	invitationRepo := &createRoomInvitationRepoStub{}
	participantRepo := &createRoomParticipantRepoStub{
		byUserID: map[shared.UserID]*participant.Participant{
			currentUserID: {ID: participant.ID(10), UserID: &currentUserID},
			invitee1:      {ID: participant.ID(20), UserID: &invitee1},
			invitee2:      {ID: participant.ID(30), UserID: &invitee2},
		},
	}
	friendshipRepo := &createRoomFriendshipRepoStub{
		findBetweenUsers: func(ctx context.Context, userID1, userID2 shared.UserID) ([]*friendship.Friendship, error) {
			return nil, friendship.ErrFriendshipNotFound
		},
	}

	uc := NewCreateChatRoomUseCase(uow, chatRoomRepo, chatMemberRepo, participantRepo, friendshipRepo, invitationRepo)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[CreateChatRoomInput]{
		Base: appShared.BaseContext{Auth: &appShared.AuthContext{UserID: currentUserID}},
		Data: CreateChatRoomInput{
			Name:      "group room",
			Type:      chatroom.Group,
			MemberIDs: []int64{int64(invitee1), int64(invitee2)},
		},
	})

	require.NoError(t, err)
	assert.Greater(t, out.ID, int64(0))
	assert.Equal(t, 1, chatRoomRepo.createCalls)
	assert.Equal(t, 1, chatMemberRepo.addCalls)
	assert.Equal(t, 2, invitationRepo.createCalls)
	require.NotNil(t, uow.tx)
	assert.True(t, uow.tx.committed)
}

package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type friendshipRepoStub struct {
	findByUserID            func(ctx context.Context, userID shared.UserID) ([]*friendship.Friendship, error)
	findAllByUserID         func(ctx context.Context, userID shared.UserID) ([]*friendship.Friendship, error)
	findPendingByUserID     func(ctx context.Context, userID shared.UserID) ([]*friendship.Friendship, error)
	create                  func(ctx context.Context, userID, friendID shared.UserID) error
	createBlocked           func(ctx context.Context, userID, friendID shared.UserID) error
	findByID                func(ctx context.Context, id int64) (*friendship.Friendship, error)
	findByUserIDAndFriendID func(ctx context.Context, userID, friendID shared.UserID) (*friendship.Friendship, error)
	findPendingByFriendID   func(ctx context.Context, userID shared.UserID) ([]*friendship.Friendship, error)
	updateStatus            func(ctx context.Context, id int64, status friendship.Status) error
	delete                  func(ctx context.Context, id int64) error
	createCalls             int
	updateCalls             int
}

func (s *friendshipRepoStub) FindByUserID(ctx context.Context, userID shared.UserID) ([]*friendship.Friendship, error) {
	if s.findByUserID != nil {
		return s.findByUserID(ctx, userID)
	}
	return nil, nil
}
func (s *friendshipRepoStub) FindAllByUserID(ctx context.Context, userID shared.UserID) ([]*friendship.Friendship, error) {
	if s.findAllByUserID != nil {
		return s.findAllByUserID(ctx, userID)
	}
	return nil, nil
}
func (s *friendshipRepoStub) FindPendingByUserID(ctx context.Context, userID shared.UserID) ([]*friendship.Friendship, error) {
	if s.findPendingByUserID != nil {
		return s.findPendingByUserID(ctx, userID)
	}
	return nil, nil
}
func (s *friendshipRepoStub) Create(ctx context.Context, userID, friendID shared.UserID) error {
	s.createCalls++
	if s.create != nil {
		return s.create(ctx, userID, friendID)
	}
	return nil
}
func (s *friendshipRepoStub) CreateBlocked(ctx context.Context, userID, friendID shared.UserID) error {
	if s.createBlocked != nil {
		return s.createBlocked(ctx, userID, friendID)
	}
	return nil
}
func (s *friendshipRepoStub) FindByID(ctx context.Context, id int64) (*friendship.Friendship, error) {
	if s.findByID != nil {
		return s.findByID(ctx, id)
	}
	return nil, friendship.ErrFriendshipNotFound
}
func (s *friendshipRepoStub) FindByUserIDAndFriendID(ctx context.Context, userID, friendID shared.UserID) (*friendship.Friendship, error) {
	if s.findByUserIDAndFriendID != nil {
		return s.findByUserIDAndFriendID(ctx, userID, friendID)
	}
	return nil, friendship.ErrFriendshipNotFound
}
func (s *friendshipRepoStub) FindBetweenUsers(context.Context, shared.UserID, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, friendship.ErrFriendshipNotFound
}
func (s *friendshipRepoStub) FindPendingByFriendID(ctx context.Context, userID shared.UserID) ([]*friendship.Friendship, error) {
	if s.findPendingByFriendID != nil {
		return s.findPendingByFriendID(ctx, userID)
	}
	return nil, nil
}
func (s *friendshipRepoStub) UpdateStatus(ctx context.Context, id int64, status friendship.Status) error {
	s.updateCalls++
	if s.updateStatus != nil {
		return s.updateStatus(ctx, id, status)
	}
	return nil
}
func (s *friendshipRepoStub) Delete(ctx context.Context, id int64) error {
	if s.delete != nil {
		return s.delete(ctx, id)
	}
	return nil
}

type friendshipUserRepoStub struct {
	users       map[shared.UserID]*user.User
	findByIDErr error
}

func (s *friendshipUserRepoStub) Create(context.Context, *user.User) (shared.UserID, error) {
	return 0, nil
}
func (s *friendshipUserRepoStub) FindByID(_ context.Context, id shared.UserID) (*user.User, error) {
	if s.findByIDErr != nil {
		return nil, s.findByIDErr
	}
	u, ok := s.users[id]
	if !ok {
		return nil, user.ErrUserNotFound
	}
	copied := *u
	copied.RoleCodes = append([]role.Code(nil), u.RoleCodes...)
	return &copied, nil
}
func (s *friendshipUserRepoStub) FindByAccountID(context.Context, shared.AccountID) (*[]user.User, error) {
	return nil, nil
}
func (s *friendshipUserRepoStub) Update(context.Context, *user.User) error { return nil }
func (s *friendshipUserRepoStub) SearchByName(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}
func (s *friendshipUserRepoStub) FindByAccountName(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}
func (s *friendshipUserRepoStub) FindByPublicID(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}

type friendshipTxStub struct {
	commitErr     error
	commitCalls   int
	rollbackCalls int
}

func (s *friendshipTxStub) Commit() error {
	s.commitCalls++
	return s.commitErr
}
func (s *friendshipTxStub) Rollback() error {
	s.rollbackCalls++
	return nil
}

type friendshipUOWStub struct {
	tx       *friendshipTxStub
	beginErr error
}

func (s *friendshipUOWStub) Begin(ctx context.Context) (context.Context, transaction.Transaction, error) {
	if s.beginErr != nil {
		return ctx, nil, s.beginErr
	}
	if s.tx == nil {
		s.tx = &friendshipTxStub{}
	}
	return ctx, s.tx, nil
}

type noopParticipantRepo struct{}

func (noopParticipantRepo) FindByID(context.Context, participant.ID) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}
func (noopParticipantRepo) FindByUserID(context.Context, shared.UserID) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}
func (noopParticipantRepo) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}
func (noopParticipantRepo) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}
func (noopParticipantRepo) Create(context.Context, *participant.Participant) error { return nil }

type noopChatRoomRepo struct{}

func (noopChatRoomRepo) FindByID(context.Context, chatroom.ID) (*chatroom.ChatRoom, error) {
	return nil, chatroom.ErrNotFound
}
func (noopChatRoomRepo) Create(context.Context, *chatroom.ChatRoom) error { return nil }
func (noopChatRoomRepo) FindDirectByParticipants(context.Context, participant.ID, participant.ID) (*chatroom.ChatRoom, error) {
	return nil, chatroom.ErrNotFound
}
func (noopChatRoomRepo) Update(context.Context, *chatroom.ChatRoom) error { return nil }
func (noopChatRoomRepo) SoftDelete(context.Context, chatroom.ID) error    { return nil }

type noopChatMemberRepo struct{}

func (noopChatMemberRepo) FindByID(context.Context, chatmember.ID) (*chatmember.ChatMember, error) {
	return nil, chatmember.ErrNotFound
}
func (noopChatMemberRepo) FindByRoomAndParticipant(context.Context, chatroom.ID, participant.ID) (*chatmember.ChatMember, error) {
	return nil, chatmember.ErrNotFound
}
func (noopChatMemberRepo) FindByRoom(context.Context, chatroom.ID) ([]*chatmember.ChatMember, error) {
	return nil, nil
}
func (noopChatMemberRepo) FindByParticipant(context.Context, participant.ID) ([]*chatmember.ChatMember, error) {
	return nil, nil
}
func (noopChatMemberRepo) Add(context.Context, *chatmember.ChatMember) error         { return nil }
func (noopChatMemberRepo) Update(context.Context, *chatmember.ChatMember) error      { return nil }
func (noopChatMemberRepo) SoftDelete(context.Context, chatmember.ID) error           { return nil }
func (noopChatMemberRepo) Remove(context.Context, chatroom.ID, participant.ID) error { return nil }

type acceptParticipantRepoStub struct {
	byUserID map[shared.UserID]*participant.Participant
}

func (s *acceptParticipantRepoStub) FindByID(context.Context, participant.ID) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *acceptParticipantRepoStub) FindByUserID(_ context.Context, userID shared.UserID) (*participant.Participant, error) {
	p, ok := s.byUserID[userID]
	if !ok {
		return nil, participant.ErrNotFound
	}
	copied := *p
	return &copied, nil
}

func (s *acceptParticipantRepoStub) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *acceptParticipantRepoStub) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *acceptParticipantRepoStub) Create(context.Context, *participant.Participant) error {
	return nil
}

type acceptChatRoomRepoStub struct {
	existingRoom       *chatroom.ChatRoom
	createCalls        int
	createdRooms       []*chatroom.ChatRoom
	softDeletedRoomIDs []chatroom.ID
}

func (s *acceptChatRoomRepoStub) FindByID(context.Context, chatroom.ID) (*chatroom.ChatRoom, error) {
	return nil, chatroom.ErrNotFound
}

func (s *acceptChatRoomRepoStub) Create(_ context.Context, room *chatroom.ChatRoom) error {
	s.createCalls++
	if room.ID == 0 {
		room.ID = chatroom.ID(700 + s.createCalls)
	}
	copied := *room
	s.createdRooms = append(s.createdRooms, &copied)
	return nil
}

func (s *acceptChatRoomRepoStub) FindDirectByParticipants(context.Context, participant.ID, participant.ID) (*chatroom.ChatRoom, error) {
	if s.existingRoom == nil {
		return nil, chatroom.ErrNotFound
	}
	copied := *s.existingRoom
	return &copied, nil
}

func (s *acceptChatRoomRepoStub) Update(context.Context, *chatroom.ChatRoom) error { return nil }
func (s *acceptChatRoomRepoStub) SoftDelete(_ context.Context, id chatroom.ID) error {
	s.softDeletedRoomIDs = append(s.softDeletedRoomIDs, id)
	return nil
}

type acceptChatMemberRepoStub struct {
	addedMembers     []*chatmember.ChatMember
	roomMembers      []*chatmember.ChatMember
	softDeletedIDs   []chatmember.ID
	softDeleteErrFor chatmember.ID
}

func (s *acceptChatMemberRepoStub) FindByID(context.Context, chatmember.ID) (*chatmember.ChatMember, error) {
	return nil, chatmember.ErrNotFound
}

func (s *acceptChatMemberRepoStub) FindByRoomAndParticipant(context.Context, chatroom.ID, participant.ID) (*chatmember.ChatMember, error) {
	return nil, chatmember.ErrNotFound
}

func (s *acceptChatMemberRepoStub) FindByRoom(context.Context, chatroom.ID) ([]*chatmember.ChatMember, error) {
	out := make([]*chatmember.ChatMember, 0, len(s.roomMembers))
	for _, member := range s.roomMembers {
		copied := *member
		out = append(out, &copied)
	}
	return out, nil
}

func (s *acceptChatMemberRepoStub) FindByParticipant(context.Context, participant.ID) ([]*chatmember.ChatMember, error) {
	return nil, nil
}

func (s *acceptChatMemberRepoStub) Add(_ context.Context, member *chatmember.ChatMember) error {
	copied := *member
	s.addedMembers = append(s.addedMembers, &copied)
	return nil
}

func (s *acceptChatMemberRepoStub) Update(context.Context, *chatmember.ChatMember) error { return nil }
func (s *acceptChatMemberRepoStub) SoftDelete(_ context.Context, id chatmember.ID) error {
	if s.softDeleteErrFor == id {
		return errors.New("soft delete failed")
	}
	s.softDeletedIDs = append(s.softDeletedIDs, id)
	return nil
}
func (s *acceptChatMemberRepoStub) Remove(context.Context, chatroom.ID, participant.ID) error {
	return nil
}

func authedInput[T any](userID shared.UserID, data T) appShared.UseCaseInput[T] {
	return appShared.UseCaseInput[T]{
		Base: appShared.BaseContext{Auth: &appShared.AuthContext{UserID: userID, Roles: []role.Code{role.User}}},
		Data: data,
	}
}

func init() {
	logger.Log = zap.NewNop()
}

func TestApplyFriendshipUseCase_RejectsSelfAndDuplicateDirections(t *testing.T) {
	uc := NewApplyFriendshipUseCase(&friendshipRepoStub{})

	err := uc.Execute(context.Background(), authedInput(501, ApplyFriendshipInput{FriendID: 501}))
	require.ErrorIs(t, err, friendship.ErrSelfFriendship)

	uc = NewApplyFriendshipUseCase(&friendshipRepoStub{
		findByUserIDAndFriendID: func(_ context.Context, userID, friendID shared.UserID) (*friendship.Friendship, error) {
			if userID == 601 && friendID == 501 {
				return &friendship.Friendship{ID: 1, UserID: 601, FriendID: 501, Status: friendship.StatusPending}, nil
			}
			return nil, friendship.ErrFriendshipNotFound
		},
	})
	err = uc.Execute(context.Background(), authedInput(501, ApplyFriendshipInput{FriendID: 601}))
	require.ErrorIs(t, err, friendship.ErrFriendshipAlreadyExists)
}

func TestApplyFriendshipUseCase_MapsDuplicateKeyAndPropagatesRepoErrors(t *testing.T) {
	repoErr := errors.New("db unavailable")
	uc := NewApplyFriendshipUseCase(&friendshipRepoStub{
		create: func(context.Context, shared.UserID, shared.UserID) error {
			return errors.New("duplicate key value violates unique constraint")
		},
	})
	err := uc.Execute(context.Background(), authedInput(501, ApplyFriendshipInput{FriendID: 601}))
	require.ErrorIs(t, err, friendship.ErrFriendshipAlreadyExists)

	uc = NewApplyFriendshipUseCase(&friendshipRepoStub{
		findByUserIDAndFriendID: func(context.Context, shared.UserID, shared.UserID) (*friendship.Friendship, error) {
			return nil, repoErr
		},
	})
	err = uc.Execute(context.Background(), authedInput(501, ApplyFriendshipInput{FriendID: 601}))
	require.ErrorIs(t, err, repoErr)
}

func TestGetFriendsAndRequestsUseCases_FilterStatusesAndSkipMissingUsers(t *testing.T) {
	friendRepo := &friendshipRepoStub{
		findByUserID: func(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
			return []*friendship.Friendship{
				{ID: 1, UserID: 501, FriendID: 601, Status: friendship.StatusAccepted, CreatedAt: time.Now()},
				{ID: 2, UserID: 501, FriendID: 602, Status: friendship.StatusPending, CreatedAt: time.Now()},
				{ID: 3, UserID: 501, FriendID: 603, Status: friendship.StatusBlocked, CreatedAt: time.Now()},
				{ID: 4, UserID: 501, FriendID: 604, Status: friendship.StatusAccepted, CreatedAt: time.Now()},
			}, nil
		},
		findPendingByFriendID: func(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
			return []*friendship.Friendship{
				{ID: 5, UserID: 605, FriendID: 501, Status: friendship.StatusPending, CreatedAt: time.Now()},
			}, nil
		},
		findPendingByUserID: func(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
			return []*friendship.Friendship{
				{ID: 2, UserID: 501, FriendID: 602, Status: friendship.StatusPending, CreatedAt: time.Now()},
			}, nil
		},
	}
	userRepo := &friendshipUserRepoStub{users: map[shared.UserID]*user.User{
		601: {ID: 601, Name: "Accepted", Avatar: "accepted.png"},
		602: {ID: 602, Name: "Pending", Avatar: "pending.png"},
		603: {ID: 603, Name: "Blocked", Avatar: "blocked.png"},
		605: {ID: 605, Name: "Requester", Avatar: "requester.png"},
	}}

	friendsOut, err := NewGetFriendsUseCase(friendRepo, userRepo).Execute(context.Background(), authedInput(501, struct{}{}))
	require.NoError(t, err)
	require.Len(t, friendsOut.Friends, 2)
	assert.Equal(t, "accepted", friendsOut.Friends[0].Status)
	assert.Equal(t, "pending", friendsOut.Friends[1].Status)

	requestsOut, err := NewGetFriendRequestsUseCase(friendRepo, userRepo).Execute(context.Background(), authedInput(501, struct{}{}))
	require.NoError(t, err)
	require.Len(t, requestsOut.Requests, 1)
	assert.Equal(t, int64(605), requestsOut.Requests[0].UserID)

	sentOut, err := NewGetSentRequestsUseCase(friendRepo, userRepo).Execute(context.Background(), authedInput(501, struct{}{}))
	require.NoError(t, err)
	require.Len(t, sentOut.Requests, 1)
	assert.Equal(t, int64(602), sentOut.Requests[0].UserID)

	blockedOut, err := NewGetBlockedUsersUseCase(friendRepo, userRepo).Execute(context.Background(), authedInput(501, struct{}{}))
	require.NoError(t, err)
	require.Len(t, blockedOut.Blocked, 1)
	assert.Equal(t, int64(603), blockedOut.Blocked[0].UserID)
	assert.Equal(t, "blocked", blockedOut.Blocked[0].Status)
}

func TestAcceptFriendshipUseCase_CreatesMutualAcceptedRowsAndCommits(t *testing.T) {
	repo := &friendshipRepoStub{}
	repo.findByID = func(context.Context, int64) (*friendship.Friendship, error) {
		return &friendship.Friendship{ID: 11, UserID: 601, FriendID: 501, Status: friendship.StatusPending}, nil
	}
	repo.findByUserIDAndFriendID = func(_ context.Context, userID, friendID shared.UserID) (*friendship.Friendship, error) {
		if userID == 501 && friendID == 601 {
			return &friendship.Friendship{ID: 12, UserID: 501, FriendID: 601, Status: friendship.StatusPending}, nil
		}
		return nil, friendship.ErrFriendshipNotFound
	}

	uow := &friendshipUOWStub{}
	uc := NewAcceptFriendshipUseCase(uow, repo, noopParticipantRepo{}, noopChatRoomRepo{}, noopChatMemberRepo{})

	err := uc.Execute(context.Background(), authedInput(501, AcceptFriendshipInput{FriendshipID: 11}))

	require.NoError(t, err)
	assert.Equal(t, 1, repo.createCalls)
	assert.Equal(t, 2, repo.updateCalls)
	assert.Equal(t, 1, uow.tx.commitCalls)
}

func TestAcceptFriendshipUseCase_CreatesDirectRoomAndOwnerMembersWhenMissing(t *testing.T) {
	repo := &friendshipRepoStub{}
	repo.findByID = func(context.Context, int64) (*friendship.Friendship, error) {
		return &friendship.Friendship{ID: 11, UserID: 601, FriendID: 501, Status: friendship.StatusPending}, nil
	}
	repo.findByUserIDAndFriendID = func(_ context.Context, userID, friendID shared.UserID) (*friendship.Friendship, error) {
		if userID == 501 && friendID == 601 {
			return &friendship.Friendship{ID: 12, UserID: 501, FriendID: 601, Status: friendship.StatusPending}, nil
		}
		return nil, friendship.ErrFriendshipNotFound
	}

	participantRepo := &acceptParticipantRepoStub{
		byUserID: map[shared.UserID]*participant.Participant{
			501: {ID: participant.ID(21), UserID: ptrUserID(501)},
			601: {ID: participant.ID(22), UserID: ptrUserID(601)},
		},
	}
	chatRoomRepo := &acceptChatRoomRepoStub{}
	chatMemberRepo := &acceptChatMemberRepoStub{}

	err := NewAcceptFriendshipUseCase(&friendshipUOWStub{}, repo, participantRepo, chatRoomRepo, chatMemberRepo).
		Execute(context.Background(), authedInput(501, AcceptFriendshipInput{FriendshipID: 11}))

	require.NoError(t, err)
	require.Len(t, chatRoomRepo.createdRooms, 1)
	assert.Equal(t, chatroom.Direct, chatRoomRepo.createdRooms[0].Type)
	assert.Equal(t, 2, chatRoomRepo.createdRooms[0].MaxMembers)
	require.Len(t, chatMemberRepo.addedMembers, 2)
	assert.Equal(t, chatRoomRepo.createdRooms[0].ID, chatMemberRepo.addedMembers[0].RoomID)
	assert.Equal(t, chatRoomRepo.createdRooms[0].ID, chatMemberRepo.addedMembers[1].RoomID)
	assert.Equal(t, participant.ID(21), chatMemberRepo.addedMembers[0].ParticipantID)
	assert.Equal(t, participant.ID(22), chatMemberRepo.addedMembers[1].ParticipantID)
	assert.Equal(t, chatmember.Owner, chatMemberRepo.addedMembers[0].Role)
	assert.Equal(t, chatmember.Owner, chatMemberRepo.addedMembers[1].Role)
}

func TestAcceptFriendshipUseCase_SkipsDirectRoomCreationWhenOneAlreadyExists(t *testing.T) {
	repo := &friendshipRepoStub{}
	repo.findByID = func(context.Context, int64) (*friendship.Friendship, error) {
		return &friendship.Friendship{ID: 11, UserID: 601, FriendID: 501, Status: friendship.StatusPending}, nil
	}
	repo.findByUserIDAndFriendID = func(_ context.Context, userID, friendID shared.UserID) (*friendship.Friendship, error) {
		if userID == 501 && friendID == 601 {
			return &friendship.Friendship{ID: 12, UserID: 501, FriendID: 601, Status: friendship.StatusPending}, nil
		}
		return nil, friendship.ErrFriendshipNotFound
	}

	participantRepo := &acceptParticipantRepoStub{
		byUserID: map[shared.UserID]*participant.Participant{
			501: {ID: participant.ID(21), UserID: ptrUserID(501)},
			601: {ID: participant.ID(22), UserID: ptrUserID(601)},
		},
	}
	chatRoomRepo := &acceptChatRoomRepoStub{
		existingRoom: &chatroom.ChatRoom{ID: chatroom.ID(88), Type: chatroom.Direct, MaxMembers: 2},
	}
	chatMemberRepo := &acceptChatMemberRepoStub{}

	err := NewAcceptFriendshipUseCase(&friendshipUOWStub{}, repo, participantRepo, chatRoomRepo, chatMemberRepo).
		Execute(context.Background(), authedInput(501, AcceptFriendshipInput{FriendshipID: 11}))

	require.NoError(t, err)
	assert.Equal(t, 0, chatRoomRepo.createCalls)
	assert.Empty(t, chatMemberRepo.addedMembers)
	assert.Equal(t, 1, repo.createCalls)
	assert.Equal(t, 2, repo.updateCalls)
}

func TestAcceptFriendshipUseCase_MapsPendingForbiddenAndCommitErrors(t *testing.T) {
	commitErr := errors.New("commit failed")
	uow := &friendshipUOWStub{tx: &friendshipTxStub{commitErr: commitErr}}
	repo := &friendshipRepoStub{
		findByID: func(context.Context, int64) (*friendship.Friendship, error) {
			return &friendship.Friendship{ID: 11, UserID: 601, FriendID: 501, Status: friendship.StatusAccepted}, nil
		},
	}
	uc := NewAcceptFriendshipUseCase(uow, repo, noopParticipantRepo{}, noopChatRoomRepo{}, noopChatMemberRepo{})

	err := uc.Execute(context.Background(), authedInput(501, AcceptFriendshipInput{FriendshipID: 11}))
	require.ErrorIs(t, err, friendship.ErrFriendshipNotPending)

	repo.findByID = func(context.Context, int64) (*friendship.Friendship, error) {
		return &friendship.Friendship{ID: 11, UserID: 601, FriendID: 999, Status: friendship.StatusPending}, nil
	}
	err = uc.Execute(context.Background(), authedInput(501, AcceptFriendshipInput{FriendshipID: 11}))
	require.ErrorIs(t, err, friendship.ErrForbidden)

	repo.findByID = func(context.Context, int64) (*friendship.Friendship, error) {
		return &friendship.Friendship{ID: 11, UserID: 601, FriendID: 501, Status: friendship.StatusPending}, nil
	}
	repo.findByUserIDAndFriendID = func(context.Context, shared.UserID, shared.UserID) (*friendship.Friendship, error) {
		return &friendship.Friendship{ID: 12, UserID: 501, FriendID: 601, Status: friendship.StatusPending}, nil
	}
	err = uc.Execute(context.Background(), authedInput(501, AcceptFriendshipInput{FriendshipID: 11}))
	require.ErrorIs(t, err, commitErr)
}

func TestAcceptFriendshipUseCase_ReturnsNotFound(t *testing.T) {
	uc := NewAcceptFriendshipUseCase(
		&friendshipUOWStub{},
		&friendshipRepoStub{
			findByID: func(context.Context, int64) (*friendship.Friendship, error) {
				return nil, friendship.ErrFriendshipNotFound
			},
		},
		noopParticipantRepo{},
		noopChatRoomRepo{},
		noopChatMemberRepo{},
	)

	err := uc.Execute(context.Background(), authedInput(501, AcceptFriendshipInput{FriendshipID: 999}))
	require.ErrorIs(t, err, friendship.ErrFriendshipNotFound)
}

func TestRemoveFriendshipUseCase_DeletesPrimaryAndReverseRows(t *testing.T) {
	deletedIDs := make([]int64, 0, 2)
	repo := &friendshipRepoStub{
		findByID: func(context.Context, int64) (*friendship.Friendship, error) {
			return &friendship.Friendship{ID: 11, UserID: 601, FriendID: 501, Status: friendship.StatusPending}, nil
		},
		findByUserIDAndFriendID: func(_ context.Context, userID, friendID shared.UserID) (*friendship.Friendship, error) {
			if userID == 501 && friendID == 601 {
				return &friendship.Friendship{ID: 12, UserID: 501, FriendID: 601, Status: friendship.StatusAccepted}, nil
			}
			return nil, friendship.ErrFriendshipNotFound
		},
		delete: func(_ context.Context, id int64) error {
			deletedIDs = append(deletedIDs, id)
			return nil
		},
	}

	_, err := NewRemoveFriendshipUseCase(&friendshipUOWStub{}, repo, noopParticipantRepo{}, noopChatRoomRepo{}, noopChatMemberRepo{}).
		Execute(context.Background(), authedInput(501, RemoveFriendshipInput{FriendshipID: 11}))
	require.NoError(t, err)
	assert.Equal(t, []int64{11, 12}, deletedIDs)
}

func TestRemoveFriendshipUseCase_SoftDeletesAcceptedDirectRoomAndMembers(t *testing.T) {
	deletedFriendshipIDs := make([]int64, 0, 2)
	repo := &friendshipRepoStub{
		findByID: func(context.Context, int64) (*friendship.Friendship, error) {
			return &friendship.Friendship{ID: 11, UserID: 601, FriendID: 501, Status: friendship.StatusAccepted}, nil
		},
		findByUserIDAndFriendID: func(_ context.Context, userID, friendID shared.UserID) (*friendship.Friendship, error) {
			if userID == 501 && friendID == 601 {
				return &friendship.Friendship{ID: 12, UserID: 501, FriendID: 601, Status: friendship.StatusAccepted}, nil
			}
			return nil, friendship.ErrFriendshipNotFound
		},
		delete: func(_ context.Context, id int64) error {
			deletedFriendshipIDs = append(deletedFriendshipIDs, id)
			return nil
		},
	}
	participantRepo := &acceptParticipantRepoStub{
		byUserID: map[shared.UserID]*participant.Participant{
			501: {ID: participant.ID(21), UserID: ptrUserID(501)},
			601: {ID: participant.ID(22), UserID: ptrUserID(601)},
		},
	}
	chatRoomRepo := &acceptChatRoomRepoStub{
		existingRoom: &chatroom.ChatRoom{ID: chatroom.ID(88), Type: chatroom.Direct},
	}
	chatMemberRepo := &acceptChatMemberRepoStub{
		roomMembers: []*chatmember.ChatMember{
			{ID: 301, RoomID: 88, ParticipantID: 21},
			{ID: 302, RoomID: 88, ParticipantID: 22},
		},
	}

	out, err := NewRemoveFriendshipUseCase(&friendshipUOWStub{}, repo, participantRepo, chatRoomRepo, chatMemberRepo).
		Execute(context.Background(), authedInput(501, RemoveFriendshipInput{FriendshipID: 11}))

	require.NoError(t, err)
	require.NotNil(t, out)
	require.NotNil(t, out.DeletedDirectRoom)
	assert.Equal(t, int64(88), out.DeletedDirectRoom.RoomID)
	assert.Equal(t, []int64{301, 302}, out.DeletedDirectRoom.MemberIDs)
	assert.Equal(t, []int64{11, 12}, deletedFriendshipIDs)
	assert.Equal(t, []chatmember.ID{301, 302}, chatMemberRepo.softDeletedIDs)
	assert.Equal(t, []chatroom.ID{88}, chatRoomRepo.softDeletedRoomIDs)
}

func TestRemoveFriendshipUseCase_PendingRejectDoesNotTouchDirectRoom(t *testing.T) {
	repo := &friendshipRepoStub{
		findByID: func(context.Context, int64) (*friendship.Friendship, error) {
			return &friendship.Friendship{ID: 11, UserID: 601, FriendID: 501, Status: friendship.StatusPending}, nil
		},
		findByUserIDAndFriendID: func(context.Context, shared.UserID, shared.UserID) (*friendship.Friendship, error) {
			return nil, friendship.ErrFriendshipNotFound
		},
	}
	participantRepo := &acceptParticipantRepoStub{
		byUserID: map[shared.UserID]*participant.Participant{
			501: {ID: participant.ID(21), UserID: ptrUserID(501)},
			601: {ID: participant.ID(22), UserID: ptrUserID(601)},
		},
	}
	chatRoomRepo := &acceptChatRoomRepoStub{
		existingRoom: &chatroom.ChatRoom{ID: chatroom.ID(88), Type: chatroom.Direct},
	}
	chatMemberRepo := &acceptChatMemberRepoStub{
		roomMembers: []*chatmember.ChatMember{
			{ID: 301, RoomID: 88, ParticipantID: 21},
			{ID: 302, RoomID: 88, ParticipantID: 22},
		},
	}

	out, err := NewRemoveFriendshipUseCase(&friendshipUOWStub{}, repo, participantRepo, chatRoomRepo, chatMemberRepo).
		Execute(context.Background(), authedInput(501, RemoveFriendshipInput{FriendshipID: 11}))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Nil(t, out.DeletedDirectRoom)
	assert.Empty(t, chatMemberRepo.softDeletedIDs)
	assert.Empty(t, chatRoomRepo.softDeletedRoomIDs)
}

func TestRemoveFriendshipUseCase_RollsBackWhenDirectRoomSoftDeleteFails(t *testing.T) {
	repo := &friendshipRepoStub{
		findByID: func(context.Context, int64) (*friendship.Friendship, error) {
			return &friendship.Friendship{ID: 11, UserID: 601, FriendID: 501, Status: friendship.StatusAccepted}, nil
		},
		findByUserIDAndFriendID: func(_ context.Context, userID, friendID shared.UserID) (*friendship.Friendship, error) {
			if userID == 501 && friendID == 601 {
				return &friendship.Friendship{ID: 12, UserID: 501, FriendID: 601, Status: friendship.StatusAccepted}, nil
			}
			return nil, friendship.ErrFriendshipNotFound
		},
	}
	participantRepo := &acceptParticipantRepoStub{
		byUserID: map[shared.UserID]*participant.Participant{
			501: {ID: participant.ID(21), UserID: ptrUserID(501)},
			601: {ID: participant.ID(22), UserID: ptrUserID(601)},
		},
	}
	chatRoomRepo := &acceptChatRoomRepoStub{
		existingRoom: &chatroom.ChatRoom{ID: chatroom.ID(88), Type: chatroom.Direct},
	}
	chatMemberRepo := &acceptChatMemberRepoStub{
		roomMembers: []*chatmember.ChatMember{
			{ID: 301, RoomID: 88, ParticipantID: 21},
			{ID: 302, RoomID: 88, ParticipantID: 22},
		},
		softDeleteErrFor: 301,
	}
	uow := &friendshipUOWStub{}

	_, err := NewRemoveFriendshipUseCase(uow, repo, participantRepo, chatRoomRepo, chatMemberRepo).
		Execute(context.Background(), authedInput(501, RemoveFriendshipInput{FriendshipID: 11}))

	require.ErrorContains(t, err, "soft delete failed")
	assert.Zero(t, uow.tx.commitCalls)
	assert.Equal(t, 1, uow.tx.rollbackCalls)
	assert.Empty(t, chatRoomRepo.softDeletedRoomIDs)
}

func TestRemoveFriendshipUseCase_MapsNotFoundAndForbidden(t *testing.T) {
	uc := NewRemoveFriendshipUseCase(&friendshipUOWStub{}, &friendshipRepoStub{
		findByID: func(context.Context, int64) (*friendship.Friendship, error) {
			return nil, friendship.ErrFriendshipNotFound
		},
	}, noopParticipantRepo{}, noopChatRoomRepo{}, noopChatMemberRepo{})
	_, err := uc.Execute(context.Background(), authedInput(501, RemoveFriendshipInput{FriendshipID: 11}))
	require.ErrorIs(t, err, friendship.ErrFriendshipNotFound)

	uc = NewRemoveFriendshipUseCase(&friendshipUOWStub{}, &friendshipRepoStub{
		findByID: func(context.Context, int64) (*friendship.Friendship, error) {
			return &friendship.Friendship{ID: 11, UserID: 601, FriendID: 602, Status: friendship.StatusAccepted}, nil
		},
	}, noopParticipantRepo{}, noopChatRoomRepo{}, noopChatMemberRepo{})
	_, err = uc.Execute(context.Background(), authedInput(501, RemoveFriendshipInput{FriendshipID: 11}))
	require.ErrorIs(t, err, friendship.ErrForbidden)
}

func TestCancelSentRequestUseCase_DeletesOnlyPendingRequestsOwnedByCaller(t *testing.T) {
	deletedIDs := make([]int64, 0, 1)
	uc := NewCancelSentRequestUseCase(&friendshipRepoStub{
		findByID: func(context.Context, int64) (*friendship.Friendship, error) {
			return &friendship.Friendship{ID: 11, UserID: 501, FriendID: 601, Status: friendship.StatusPending}, nil
		},
		delete: func(_ context.Context, id int64) error {
			deletedIDs = append(deletedIDs, id)
			return nil
		},
	})

	err := uc.Execute(context.Background(), authedInput(501, CancelSentRequestInput{FriendshipID: 11}))
	require.NoError(t, err)
	assert.Equal(t, []int64{11}, deletedIDs)

	uc = NewCancelSentRequestUseCase(&friendshipRepoStub{
		findByID: func(context.Context, int64) (*friendship.Friendship, error) {
			return &friendship.Friendship{ID: 11, UserID: 601, FriendID: 501, Status: friendship.StatusPending}, nil
		},
	})
	err = uc.Execute(context.Background(), authedInput(501, CancelSentRequestInput{FriendshipID: 11}))
	require.ErrorIs(t, err, friendship.ErrForbidden)

	uc = NewCancelSentRequestUseCase(&friendshipRepoStub{
		findByID: func(context.Context, int64) (*friendship.Friendship, error) {
			return &friendship.Friendship{ID: 11, UserID: 501, FriendID: 601, Status: friendship.StatusAccepted}, nil
		},
	})
	err = uc.Execute(context.Background(), authedInput(501, CancelSentRequestInput{FriendshipID: 11}))
	require.ErrorIs(t, err, friendship.ErrFriendshipNotPending)

	uc = NewCancelSentRequestUseCase(&friendshipRepoStub{
		findByID: func(context.Context, int64) (*friendship.Friendship, error) {
			return nil, friendship.ErrFriendshipNotFound
		},
	})
	err = uc.Execute(context.Background(), authedInput(501, CancelSentRequestInput{FriendshipID: 11}))
	require.ErrorIs(t, err, friendship.ErrFriendshipNotFound)
}

func TestBlockUserUseCase_CreatesOrUpdatesBlockedRelationships(t *testing.T) {
	createBlockedCalls := make([][2]shared.UserID, 0, 1)
	deletedIDs := make([]int64, 0, 1)
	uc := NewBlockUserUseCase(&friendshipRepoStub{
		findByUserIDAndFriendID: func(_ context.Context, userID, friendID shared.UserID) (*friendship.Friendship, error) {
			if userID == 601 && friendID == 501 {
				return &friendship.Friendship{ID: 22, UserID: 601, FriendID: 501, Status: friendship.StatusPending}, nil
			}
			return nil, friendship.ErrFriendshipNotFound
		},
		createBlocked: func(_ context.Context, userID, friendID shared.UserID) error {
			createBlockedCalls = append(createBlockedCalls, [2]shared.UserID{userID, friendID})
			return nil
		},
		delete: func(_ context.Context, id int64) error {
			deletedIDs = append(deletedIDs, id)
			return nil
		},
	})

	err := uc.Execute(context.Background(), authedInput(501, BlockUserInput{TargetUserID: 601}))
	require.NoError(t, err)
	assert.Equal(t, [][2]shared.UserID{{501, 601}}, createBlockedCalls)
	assert.Equal(t, []int64{22}, deletedIDs)

	updateCalls := make([]int64, 0, 1)
	uc = NewBlockUserUseCase(&friendshipRepoStub{
		findByUserIDAndFriendID: func(_ context.Context, userID, friendID shared.UserID) (*friendship.Friendship, error) {
			if userID == 501 && friendID == 601 {
				return &friendship.Friendship{ID: 11, UserID: 501, FriendID: 601, Status: friendship.StatusAccepted}, nil
			}
			if userID == 601 && friendID == 501 {
				return &friendship.Friendship{ID: 12, UserID: 601, FriendID: 501, Status: friendship.StatusAccepted}, nil
			}
			return nil, friendship.ErrFriendshipNotFound
		},
		updateStatus: func(_ context.Context, id int64, status friendship.Status) error {
			updateCalls = append(updateCalls, id)
			assert.Equal(t, friendship.StatusBlocked, status)
			return nil
		},
		delete: func(_ context.Context, id int64) error {
			deletedIDs = append(deletedIDs, id)
			return nil
		},
	})

	err = uc.Execute(context.Background(), authedInput(501, BlockUserInput{TargetUserID: 601}))
	require.NoError(t, err)
	assert.Equal(t, []int64{11}, updateCalls)
	assert.Contains(t, deletedIDs, int64(12))

	uc = NewBlockUserUseCase(&friendshipRepoStub{
		findByUserIDAndFriendID: func(_ context.Context, userID, friendID shared.UserID) (*friendship.Friendship, error) {
			if userID == 501 && friendID == 601 {
				return &friendship.Friendship{ID: 11, UserID: 501, FriendID: 601, Status: friendship.StatusBlocked}, nil
			}
			return nil, friendship.ErrFriendshipNotFound
		},
	})
	err = uc.Execute(context.Background(), authedInput(501, BlockUserInput{TargetUserID: 601}))
	require.ErrorIs(t, err, friendship.ErrAlreadyBlocked)
}

func TestUnblockUserUseCase_DeletesBlockedRelationshipsOnly(t *testing.T) {
	deletedIDs := make([]int64, 0, 1)
	uc := NewUnblockUserUseCase(&friendshipRepoStub{
		findByUserIDAndFriendID: func(_ context.Context, userID, friendID shared.UserID) (*friendship.Friendship, error) {
			return &friendship.Friendship{ID: 11, UserID: userID, FriendID: friendID, Status: friendship.StatusBlocked}, nil
		},
		delete: func(_ context.Context, id int64) error {
			deletedIDs = append(deletedIDs, id)
			return nil
		},
	})

	err := uc.Execute(context.Background(), authedInput(501, UnblockUserInput{TargetUserID: 601}))
	require.NoError(t, err)
	assert.Equal(t, []int64{11}, deletedIDs)

	uc = NewUnblockUserUseCase(&friendshipRepoStub{
		findByUserIDAndFriendID: func(_ context.Context, userID, friendID shared.UserID) (*friendship.Friendship, error) {
			return &friendship.Friendship{ID: 11, UserID: userID, FriendID: friendID, Status: friendship.StatusAccepted}, nil
		},
	})
	err = uc.Execute(context.Background(), authedInput(501, UnblockUserInput{TargetUserID: 601}))
	require.ErrorIs(t, err, friendship.ErrNotBlocked)

	uc = NewUnblockUserUseCase(&friendshipRepoStub{
		findByUserIDAndFriendID: func(context.Context, shared.UserID, shared.UserID) (*friendship.Friendship, error) {
			return nil, friendship.ErrFriendshipNotFound
		},
	})
	err = uc.Execute(context.Background(), authedInput(501, UnblockUserInput{TargetUserID: 601}))
	require.ErrorIs(t, err, friendship.ErrNotBlocked)
}

func ptrUserID(id shared.UserID) *shared.UserID {
	return &id
}

package friendship

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	friendshipUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/friendship/usecase"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	domainfriendship "github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type handlerFriendshipRepoStub struct {
	findByUserID            func(ctx context.Context, userID shared.UserID) ([]*domainfriendship.Friendship, error)
	findAllByUserID         func(ctx context.Context, userID shared.UserID) ([]*domainfriendship.Friendship, error)
	findPendingByFriendID   func(ctx context.Context, friendID shared.UserID) ([]*domainfriendship.Friendship, error)
	findByUserIDAndFriendID func(ctx context.Context, userID, friendID shared.UserID) (*domainfriendship.Friendship, error)
	findByID                func(ctx context.Context, id int64) (*domainfriendship.Friendship, error)
	create                  func(ctx context.Context, userID, friendID shared.UserID) error
	createBlocked           func(ctx context.Context, userID, friendID shared.UserID) error
	updateStatus            func(ctx context.Context, id int64, status domainfriendship.Status) error
	delete                  func(ctx context.Context, id int64) error
}

func (s *handlerFriendshipRepoStub) FindByUserID(ctx context.Context, userID shared.UserID) ([]*domainfriendship.Friendship, error) {
	if s.findByUserID != nil {
		return s.findByUserID(ctx, userID)
	}
	return nil, nil
}
func (s *handlerFriendshipRepoStub) FindAllByUserID(ctx context.Context, userID shared.UserID) ([]*domainfriendship.Friendship, error) {
	if s.findAllByUserID != nil {
		return s.findAllByUserID(ctx, userID)
	}
	return nil, nil
}
func (s *handlerFriendshipRepoStub) FindPendingByUserID(context.Context, shared.UserID) ([]*domainfriendship.Friendship, error) {
	return nil, nil
}
func (s *handlerFriendshipRepoStub) Create(ctx context.Context, userID, friendID shared.UserID) error {
	if s.create != nil {
		return s.create(ctx, userID, friendID)
	}
	return nil
}
func (s *handlerFriendshipRepoStub) CreateBlocked(ctx context.Context, userID, friendID shared.UserID) error {
	if s.createBlocked != nil {
		return s.createBlocked(ctx, userID, friendID)
	}
	return nil
}
func (s *handlerFriendshipRepoStub) FindByID(ctx context.Context, id int64) (*domainfriendship.Friendship, error) {
	if s.findByID != nil {
		return s.findByID(ctx, id)
	}
	return nil, domainfriendship.ErrFriendshipNotFound
}
func (s *handlerFriendshipRepoStub) FindByUserIDAndFriendID(ctx context.Context, userID, friendID shared.UserID) (*domainfriendship.Friendship, error) {
	if s.findByUserIDAndFriendID != nil {
		return s.findByUserIDAndFriendID(ctx, userID, friendID)
	}
	return nil, domainfriendship.ErrFriendshipNotFound
}
func (s *handlerFriendshipRepoStub) FindBetweenUsers(context.Context, shared.UserID, shared.UserID) ([]*domainfriendship.Friendship, error) {
	return nil, domainfriendship.ErrFriendshipNotFound
}
func (s *handlerFriendshipRepoStub) FindPendingByFriendID(ctx context.Context, friendID shared.UserID) ([]*domainfriendship.Friendship, error) {
	if s.findPendingByFriendID != nil {
		return s.findPendingByFriendID(ctx, friendID)
	}
	return nil, nil
}
func (s *handlerFriendshipRepoStub) UpdateStatus(ctx context.Context, id int64, status domainfriendship.Status) error {
	if s.updateStatus != nil {
		return s.updateStatus(ctx, id, status)
	}
	return nil
}
func (s *handlerFriendshipRepoStub) Delete(ctx context.Context, id int64) error {
	if s.delete != nil {
		return s.delete(ctx, id)
	}
	return nil
}

type handlerTxStub struct{}

func (handlerTxStub) Commit() error   { return nil }
func (handlerTxStub) Rollback() error { return nil }

type handlerUOWStub struct{}

func (handlerUOWStub) Begin(ctx context.Context) (context.Context, transaction.Transaction, error) {
	return ctx, handlerTxStub{}, nil
}

type handlerParticipantRepoStub struct{}

func (handlerParticipantRepoStub) FindByID(context.Context, participant.ID) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}
func (handlerParticipantRepoStub) FindByUserID(context.Context, shared.UserID) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}
func (handlerParticipantRepoStub) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}
func (handlerParticipantRepoStub) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}
func (handlerParticipantRepoStub) Create(context.Context, *participant.Participant) error { return nil }

type handlerChatRoomRepoStub struct{}

func (handlerChatRoomRepoStub) FindByID(context.Context, chatroom.ID) (*chatroom.ChatRoom, error) {
	return nil, chatroom.ErrNotFound
}
func (handlerChatRoomRepoStub) Create(context.Context, *chatroom.ChatRoom) error { return nil }
func (handlerChatRoomRepoStub) FindDirectByParticipants(context.Context, participant.ID, participant.ID) (*chatroom.ChatRoom, error) {
	return nil, chatroom.ErrNotFound
}
func (handlerChatRoomRepoStub) Update(context.Context, *chatroom.ChatRoom) error { return nil }
func (handlerChatRoomRepoStub) SoftDelete(context.Context, chatroom.ID) error    { return nil }

type handlerChatMemberRepoStub struct{}

func (handlerChatMemberRepoStub) FindByID(context.Context, chatmember.ID) (*chatmember.ChatMember, error) {
	return nil, chatmember.ErrNotFound
}
func (handlerChatMemberRepoStub) FindByRoomAndParticipant(context.Context, chatroom.ID, participant.ID) (*chatmember.ChatMember, error) {
	return nil, chatmember.ErrNotFound
}
func (handlerChatMemberRepoStub) FindByRoom(context.Context, chatroom.ID) ([]*chatmember.ChatMember, error) {
	return nil, nil
}
func (handlerChatMemberRepoStub) FindByParticipant(context.Context, participant.ID) ([]*chatmember.ChatMember, error) {
	return nil, nil
}
func (handlerChatMemberRepoStub) Add(context.Context, *chatmember.ChatMember) error    { return nil }
func (handlerChatMemberRepoStub) Update(context.Context, *chatmember.ChatMember) error { return nil }
func (handlerChatMemberRepoStub) SoftDelete(context.Context, chatmember.ID) error      { return nil }
func (handlerChatMemberRepoStub) Remove(context.Context, chatroom.ID, participant.ID) error {
	return nil
}

func friendshipTestRouter(handler *FriendshipHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.AuthContextKey, &appShared.AuthContext{UserID: 501})
		c.Next()
	})
	router.Use(middleware.ContextMiddleware())
	handler.RegisterFriendshipRoutes(router.Group(""))
	return router
}

func TestFriendshipHandler_ApplyRejectsInvalidPayload(t *testing.T) {
	handler := NewFriendshipHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/friends/apply", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	friendshipTestRouter(handler).ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "INVALID_REQUEST", body.Error.Code)
}

func TestFriendshipHandler_AcceptRejectsInvalidID(t *testing.T) {
	handler := NewFriendshipHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/friends/not-a-number/accept", nil)
	resp := httptest.NewRecorder()

	friendshipTestRouter(handler).ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "INVALID_ID")
}

func TestFriendshipHandler_MapsConflictAndSelfTargetErrors(t *testing.T) {
	repo := &handlerFriendshipRepoStub{
		findByUserIDAndFriendID: func(_ context.Context, userID, friendID shared.UserID) (*domainfriendship.Friendship, error) {
			if userID == 601 && friendID == 501 {
				return &domainfriendship.Friendship{ID: 1, UserID: 601, FriendID: 501, Status: domainfriendship.StatusPending}, nil
			}
			return nil, domainfriendship.ErrFriendshipNotFound
		},
		findByID: func(context.Context, int64) (*domainfriendship.Friendship, error) {
			return &domainfriendship.Friendship{ID: 2, UserID: 601, FriendID: 501, Status: domainfriendship.StatusPending}, nil
		},
	}
	handler := NewFriendshipHandler(
		friendshipUseCase.NewGetFriendsUseCase(repo, &handlerUserRepoStub{}),
		friendshipUseCase.NewGetBlockedUsersUseCase(repo, &handlerUserRepoStub{}),
		friendshipUseCase.NewApplyFriendshipUseCase(repo),
		friendshipUseCase.NewAcceptFriendshipUseCase(handlerUOWStub{}, repo, handlerParticipantRepoStub{}, handlerChatRoomRepoStub{}, handlerChatMemberRepoStub{}),
		friendshipUseCase.NewGetFriendRequestsUseCase(repo, &handlerUserRepoStub{}),
		friendshipUseCase.NewRemoveFriendshipUseCase(handlerUOWStub{}, repo, handlerParticipantRepoStub{}, handlerChatRoomRepoStub{}, handlerChatMemberRepoStub{}),
		friendshipUseCase.NewGetSentRequestsUseCase(repo, &handlerUserRepoStub{}),
		friendshipUseCase.NewCancelSentRequestUseCase(repo),
		friendshipUseCase.NewBlockUserUseCase(repo),
		friendshipUseCase.NewUnblockUserUseCase(repo),
	)
	router := friendshipTestRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/friends/apply", bytes.NewBufferString(`{"friend_id":601}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusConflict, resp.Code)
	assert.Contains(t, resp.Body.String(), "FRIENDSHIP_EXISTS")

	req = httptest.NewRequest(http.MethodPost, "/friends/apply", bytes.NewBufferString(`{"friend_id":501}`))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "INVALID_FRIENDSHIP_TARGET")
}

func TestFriendshipHandler_RemoveCancelAndBlockRejectInvalidIDs(t *testing.T) {
	handler := NewFriendshipHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := friendshipTestRouter(handler)

	cases := []struct {
		method string
		path   string
	}{
		{method: http.MethodDelete, path: "/friends/not-a-number"},
		{method: http.MethodDelete, path: "/friends/sent/not-a-number"},
		{method: http.MethodPost, path: "/block/not-a-number"},
		{method: http.MethodDelete, path: "/block/not-a-number"},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		require.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "INVALID_ID")
	}
}

func TestFriendshipHandler_MapsRemoveCancelAndBlockErrors(t *testing.T) {
	repo := &handlerFriendshipRepoStub{
		findByID: func(_ context.Context, id int64) (*domainfriendship.Friendship, error) {
			switch id {
			case 11:
				return nil, domainfriendship.ErrFriendshipNotFound
			case 12:
				return &domainfriendship.Friendship{ID: 12, UserID: 601, FriendID: 501, Status: domainfriendship.StatusAccepted}, nil
			case 13:
				return &domainfriendship.Friendship{ID: 13, UserID: 501, FriendID: 601, Status: domainfriendship.StatusAccepted}, nil
			default:
				return nil, domainfriendship.ErrFriendshipNotFound
			}
		},
		findByUserIDAndFriendID: func(_ context.Context, userID, friendID shared.UserID) (*domainfriendship.Friendship, error) {
			if userID == 501 && friendID == 601 {
				return &domainfriendship.Friendship{ID: 21, UserID: 501, FriendID: 601, Status: domainfriendship.StatusBlocked}, nil
			}
			return nil, domainfriendship.ErrFriendshipNotFound
		},
	}
	handler := NewFriendshipHandler(
		friendshipUseCase.NewGetFriendsUseCase(repo, &handlerUserRepoStub{}),
		friendshipUseCase.NewGetBlockedUsersUseCase(repo, &handlerUserRepoStub{}),
		friendshipUseCase.NewApplyFriendshipUseCase(repo),
		friendshipUseCase.NewAcceptFriendshipUseCase(handlerUOWStub{}, repo, handlerParticipantRepoStub{}, handlerChatRoomRepoStub{}, handlerChatMemberRepoStub{}),
		friendshipUseCase.NewGetFriendRequestsUseCase(repo, &handlerUserRepoStub{}),
		friendshipUseCase.NewRemoveFriendshipUseCase(handlerUOWStub{}, repo, handlerParticipantRepoStub{}, handlerChatRoomRepoStub{}, handlerChatMemberRepoStub{}),
		friendshipUseCase.NewGetSentRequestsUseCase(repo, &handlerUserRepoStub{}),
		friendshipUseCase.NewCancelSentRequestUseCase(repo),
		friendshipUseCase.NewBlockUserUseCase(repo),
		friendshipUseCase.NewUnblockUserUseCase(repo),
	)
	router := friendshipTestRouter(handler)

	req := httptest.NewRequest(http.MethodDelete, "/friends/11", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusNotFound, resp.Code)
	assert.Contains(t, resp.Body.String(), "NOT_FOUND")

	req = httptest.NewRequest(http.MethodDelete, "/friends/sent/12", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusForbidden, resp.Code)
	assert.Contains(t, resp.Body.String(), "FORBIDDEN")

	req = httptest.NewRequest(http.MethodDelete, "/friends/sent/13", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "FRIENDSHIP_NOT_PENDING")

	req = httptest.NewRequest(http.MethodPost, "/block/601", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusConflict, resp.Code)
	assert.Contains(t, resp.Body.String(), "ALREADY_BLOCKED")

	req = httptest.NewRequest(http.MethodDelete, "/block/602", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "NOT_BLOCKED")
}

type handlerUserRepoStub struct{}

func (handlerUserRepoStub) Create(context.Context, *user.User) (shared.UserID, error) { return 0, nil }
func (handlerUserRepoStub) FindByID(context.Context, shared.UserID) (*user.User, error) {
	return &user.User{ID: 501}, nil
}
func (handlerUserRepoStub) FindByAccountID(context.Context, shared.AccountID) (*[]user.User, error) {
	return nil, nil
}
func (handlerUserRepoStub) Update(context.Context, *user.User) error { return nil }
func (handlerUserRepoStub) SearchByName(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}
func (handlerUserRepoStub) FindByAccountName(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}
func (handlerUserRepoStub) FindByPublicID(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}

type handlerOverviewUserRepoStub struct {
	users map[shared.UserID]*user.User
}

func (s handlerOverviewUserRepoStub) Create(context.Context, *user.User) (shared.UserID, error) {
	return 0, nil
}
func (s handlerOverviewUserRepoStub) FindByID(_ context.Context, id shared.UserID) (*user.User, error) {
	u, ok := s.users[id]
	if !ok {
		return nil, user.ErrUserNotFound
	}
	cloned := *u
	return &cloned, nil
}
func (s handlerOverviewUserRepoStub) FindByAccountID(context.Context, shared.AccountID) (*[]user.User, error) {
	return nil, nil
}
func (s handlerOverviewUserRepoStub) Update(context.Context, *user.User) error { return nil }
func (s handlerOverviewUserRepoStub) SearchByName(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}
func (s handlerOverviewUserRepoStub) FindByAccountName(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}
func (s handlerOverviewUserRepoStub) FindByPublicID(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}

func TestFriendshipHandler_GetOverviewReturnsAggregatedLists(t *testing.T) {
	now := time.Date(2026, 1, 1, 9, 30, 0, 0, time.UTC)
	repo := &handlerFriendshipRepoStub{
		findByUserID: func(_ context.Context, userID shared.UserID) ([]*domainfriendship.Friendship, error) {
			require.Equal(t, shared.UserID(501), userID)
			return []*domainfriendship.Friendship{
				{ID: 1, UserID: 501, FriendID: 601, Status: domainfriendship.StatusAccepted, CreatedAt: now},
				{ID: 3, UserID: 501, FriendID: 603, Status: domainfriendship.StatusBlocked, CreatedAt: now.Add(time.Minute)},
			}, nil
		},
		findAllByUserID: func(_ context.Context, userID shared.UserID) ([]*domainfriendship.Friendship, error) {
			require.Equal(t, shared.UserID(501), userID)
			return []*domainfriendship.Friendship{
				{ID: 1, UserID: 501, FriendID: 601, Status: domainfriendship.StatusAccepted, CreatedAt: now},
				{ID: 3, UserID: 501, FriendID: 603, Status: domainfriendship.StatusBlocked, CreatedAt: now.Add(time.Minute)},
			}, nil
		},
		findPendingByFriendID: func(_ context.Context, friendID shared.UserID) ([]*domainfriendship.Friendship, error) {
			require.Equal(t, shared.UserID(501), friendID)
			return []*domainfriendship.Friendship{
				{ID: 2, UserID: 602, FriendID: 501, Status: domainfriendship.StatusPending, CreatedAt: now.Add(2 * time.Minute)},
			}, nil
		},
		findByUserIDAndFriendID: func(_ context.Context, userID, friendID shared.UserID) (*domainfriendship.Friendship, error) {
			if userID == 501 && friendID == 602 {
				return nil, domainfriendship.ErrFriendshipNotFound
			}
			return nil, domainfriendship.ErrFriendshipNotFound
		},
	}
	userRepo := handlerOverviewUserRepoStub{
		users: map[shared.UserID]*user.User{
			601: {ID: 601, Name: "Accepted Friend", Avatar: "accepted.png"},
			602: {ID: 602, Name: "Pending Request", Avatar: "pending.png"},
			603: {ID: 603, Name: "Blocked User", Avatar: "blocked.png"},
		},
	}
	handler := NewFriendshipHandler(
		friendshipUseCase.NewGetFriendsUseCase(repo, userRepo),
		friendshipUseCase.NewGetBlockedUsersUseCase(repo, userRepo),
		friendshipUseCase.NewApplyFriendshipUseCase(repo),
		friendshipUseCase.NewAcceptFriendshipUseCase(handlerUOWStub{}, repo, handlerParticipantRepoStub{}, handlerChatRoomRepoStub{}, handlerChatMemberRepoStub{}),
		friendshipUseCase.NewGetFriendRequestsUseCase(repo, userRepo),
		friendshipUseCase.NewRemoveFriendshipUseCase(handlerUOWStub{}, repo, handlerParticipantRepoStub{}, handlerChatRoomRepoStub{}, handlerChatMemberRepoStub{}),
		friendshipUseCase.NewGetSentRequestsUseCase(repo, userRepo),
		friendshipUseCase.NewCancelSentRequestUseCase(repo),
		friendshipUseCase.NewBlockUserUseCase(repo),
		friendshipUseCase.NewUnblockUserUseCase(repo),
	)

	req := httptest.NewRequest(http.MethodGet, "/friends/overview", nil)
	resp := httptest.NewRecorder()
	friendshipTestRouter(handler).ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body FriendsOverviewResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Len(t, body.Friends, 1)
	assert.Equal(t, "Accepted Friend", body.Friends[0].Name)
	require.Len(t, body.Requests, 1)
	assert.Equal(t, "Pending Request", body.Requests[0].Name)
	require.Len(t, body.Blocked, 1)
	assert.Equal(t, "Blocked User", body.Blocked[0].Name)
	assert.Equal(t, "blocked", body.Blocked[0].Status)
}

func TestFriendshipHandler_GetOverviewPropagatesUseCaseErrors(t *testing.T) {
	repo := &handlerFriendshipRepoStub{
		findByUserID: func(context.Context, shared.UserID) ([]*domainfriendship.Friendship, error) {
			return []*domainfriendship.Friendship{}, nil
		},
		findAllByUserID: func(context.Context, shared.UserID) ([]*domainfriendship.Friendship, error) {
			return []*domainfriendship.Friendship{}, nil
		},
		findPendingByFriendID: func(context.Context, shared.UserID) ([]*domainfriendship.Friendship, error) {
			return nil, errors.New("requests unavailable")
		},
	}
	userRepo := handlerOverviewUserRepoStub{users: map[shared.UserID]*user.User{}}
	handler := NewFriendshipHandler(
		friendshipUseCase.NewGetFriendsUseCase(repo, userRepo),
		friendshipUseCase.NewGetBlockedUsersUseCase(repo, userRepo),
		friendshipUseCase.NewApplyFriendshipUseCase(repo),
		friendshipUseCase.NewAcceptFriendshipUseCase(handlerUOWStub{}, repo, handlerParticipantRepoStub{}, handlerChatRoomRepoStub{}, handlerChatMemberRepoStub{}),
		friendshipUseCase.NewGetFriendRequestsUseCase(repo, userRepo),
		friendshipUseCase.NewRemoveFriendshipUseCase(handlerUOWStub{}, repo, handlerParticipantRepoStub{}, handlerChatRoomRepoStub{}, handlerChatMemberRepoStub{}),
		friendshipUseCase.NewGetSentRequestsUseCase(repo, userRepo),
		friendshipUseCase.NewCancelSentRequestUseCase(repo),
		friendshipUseCase.NewBlockUserUseCase(repo),
		friendshipUseCase.NewUnblockUserUseCase(repo),
	)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.AuthContextKey, &appShared.AuthContext{UserID: 501})
		c.Next()
		if len(c.Errors) > 0 && !c.Writer.Written() {
			c.JSON(http.StatusInternalServerError, gin.H{"error": c.Errors.Last().Error()})
		}
	})
	router.Use(middleware.ContextMiddleware())
	handler.RegisterFriendshipRoutes(router.Group(""))

	req := httptest.NewRequest(http.MethodGet, "/friends/overview", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Contains(t, resp.Body.String(), "requests unavailable")
}

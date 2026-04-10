package usecase

import (
	"context"
	"errors"
	"testing"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type searchUsersRepoStub struct {
	searchByName      func(ctx context.Context, keyword string, limit, offset int) ([]*user.UserSearchResult, error)
	findByAccountName func(ctx context.Context, accountName string, limit, offset int) ([]*user.UserSearchResult, error)
	findByPublicID    func(ctx context.Context, publicID string, limit, offset int) ([]*user.UserSearchResult, error)
	lastLimit         int
	lastOffset        int
}

func (s *searchUsersRepoStub) Create(context.Context, *user.User) (shared.UserID, error) {
	return 0, nil
}
func (s *searchUsersRepoStub) FindByID(context.Context, shared.UserID) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (s *searchUsersRepoStub) FindByAccountID(context.Context, shared.AccountID) (*[]user.User, error) {
	return nil, nil
}
func (s *searchUsersRepoStub) Update(context.Context, *user.User) error { return nil }
func (s *searchUsersRepoStub) SearchByName(ctx context.Context, keyword string, limit, offset int) ([]*user.UserSearchResult, error) {
	s.lastLimit = limit
	s.lastOffset = offset
	if s.searchByName != nil {
		return s.searchByName(ctx, keyword, limit, offset)
	}
	return nil, nil
}
func (s *searchUsersRepoStub) FindByAccountName(ctx context.Context, accountName string, limit, offset int) ([]*user.UserSearchResult, error) {
	s.lastLimit = limit
	s.lastOffset = offset
	if s.findByAccountName != nil {
		return s.findByAccountName(ctx, accountName, limit, offset)
	}
	return nil, nil
}
func (s *searchUsersRepoStub) FindByPublicID(ctx context.Context, publicID string, limit, offset int) ([]*user.UserSearchResult, error) {
	s.lastLimit = limit
	s.lastOffset = offset
	if s.findByPublicID != nil {
		return s.findByPublicID(ctx, publicID, limit, offset)
	}
	return nil, nil
}

type searchFriendshipRepoStub struct {
	findAllByUserID func(ctx context.Context, userID shared.UserID) ([]*friendship.Friendship, error)
}

func (s *searchFriendshipRepoStub) FindByUserID(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}
func (s *searchFriendshipRepoStub) FindAllByUserID(ctx context.Context, userID shared.UserID) ([]*friendship.Friendship, error) {
	if s.findAllByUserID != nil {
		return s.findAllByUserID(ctx, userID)
	}
	return nil, nil
}
func (s *searchFriendshipRepoStub) FindPendingByUserID(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}
func (s *searchFriendshipRepoStub) Create(context.Context, shared.UserID, shared.UserID) error {
	return nil
}
func (s *searchFriendshipRepoStub) CreateBlocked(context.Context, shared.UserID, shared.UserID) error {
	return nil
}
func (s *searchFriendshipRepoStub) FindByID(context.Context, int64) (*friendship.Friendship, error) {
	return nil, friendship.ErrFriendshipNotFound
}
func (s *searchFriendshipRepoStub) FindByUserIDAndFriendID(context.Context, shared.UserID, shared.UserID) (*friendship.Friendship, error) {
	return nil, friendship.ErrFriendshipNotFound
}
func (s *searchFriendshipRepoStub) FindBetweenUsers(context.Context, shared.UserID, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, friendship.ErrFriendshipNotFound
}
func (s *searchFriendshipRepoStub) FindPendingByFriendID(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
	return nil, nil
}
func (s *searchFriendshipRepoStub) UpdateStatus(context.Context, int64, friendship.Status) error {
	return nil
}
func (s *searchFriendshipRepoStub) Delete(context.Context, int64) error { return nil }

func TestSearchUsersUseCase_RequiresExactlyOneSearchParam(t *testing.T) {
	uc := NewSearchUsersUseCase(&searchUsersRepoStub{}, &searchFriendshipRepoStub{})

	_, err := uc.Execute(context.Background(), appShared.UseCaseInput[SearchUsersInput]{})
	require.ErrorIs(t, err, ErrInvalidSearchInput)

	_, err = uc.Execute(context.Background(), appShared.UseCaseInput[SearchUsersInput]{
		Data: SearchUsersInput{Name: "hiro", Account: "hiro_account"},
	})
	require.ErrorIs(t, err, ErrInvalidSearchInput)
}

func TestSearchUsersUseCase_FiltersCurrentUserAndMapsFriendshipStatus(t *testing.T) {
	userRepo := &searchUsersRepoStub{
		searchByName: func(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
			return []*user.UserSearchResult{
				{ID: 501, Name: "Hiro", AccountName: "hiro_account", PublicID: "hiro-public"},
				{ID: 601, Name: "Mina", AccountName: "mina_account", PublicID: "mina-public"},
				{ID: 602, Name: "Luna", AccountName: "luna_account", PublicID: "luna-public"},
				{ID: 603, Name: "Blocked Me", AccountName: "blocked_account", PublicID: "blocked-public"},
				{ID: 604, Name: "I Blocked", AccountName: "i_blocked_account", PublicID: "i-blocked-public"},
			}, nil
		},
	}
	friendshipRepo := &searchFriendshipRepoStub{
		findAllByUserID: func(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
			return []*friendship.Friendship{
				{UserID: 501, FriendID: 601, Status: friendship.StatusPending},
				{UserID: 602, FriendID: 501, Status: friendship.StatusAccepted},
				{UserID: 603, FriendID: 501, Status: friendship.StatusBlocked},
				{UserID: 501, FriendID: 604, Status: friendship.StatusBlocked},
			}, nil
		},
	}

	uc := NewSearchUsersUseCase(userRepo, friendshipRepo)
	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[SearchUsersInput]{
		Base: appShared.BaseContext{Auth: &appShared.AuthContext{UserID: 501, Roles: []role.Code{role.User}}},
		Data: SearchUsersInput{Name: "a"},
	})

	require.NoError(t, err)
	require.Len(t, out.Users, 3)
	assert.Equal(t, shared.UserID(601), out.Users[0].ID)
	assert.Equal(t, "pending", *out.Users[0].FriendshipStatus)
	assert.Equal(t, shared.UserID(602), out.Users[1].ID)
	assert.Equal(t, "accepted", *out.Users[1].FriendshipStatus)
	assert.Equal(t, shared.UserID(604), out.Users[2].ID)
	assert.Equal(t, "blocked", *out.Users[2].FriendshipStatus)
	require.NotNil(t, out.Users[2].BlockedBy)
	assert.Equal(t, "me", *out.Users[2].BlockedBy)
}

func TestSearchUsersUseCase_UsesDefaultLimitAndOffsetFallback(t *testing.T) {
	userRepo := &searchUsersRepoStub{
		findByAccountName: func(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
			return []*user.UserSearchResult{}, nil
		},
	}
	uc := NewSearchUsersUseCase(userRepo, &searchFriendshipRepoStub{})

	_, err := uc.Execute(context.Background(), appShared.UseCaseInput[SearchUsersInput]{
		Base: appShared.BaseContext{Auth: &appShared.AuthContext{UserID: 501, Roles: []role.Code{role.User}}},
		Data: SearchUsersInput{Account: "hiro_account", Limit: 0, Offset: -5},
	})

	require.NoError(t, err)
	assert.Equal(t, defaultSearchLimit, userRepo.lastLimit)
	assert.Equal(t, 0, userRepo.lastOffset)
}

func TestSearchUsersUseCase_PropagatesRepoErrorsAndIgnoresFriendshipLookupFailure(t *testing.T) {
	expectedErr := errors.New("search failed")
	uc := NewSearchUsersUseCase(&searchUsersRepoStub{
		findByPublicID: func(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
			return nil, expectedErr
		},
	}, &searchFriendshipRepoStub{
		findAllByUserID: func(context.Context, shared.UserID) ([]*friendship.Friendship, error) {
			return nil, errors.New("friendship lookup failed")
		},
	})

	_, err := uc.Execute(context.Background(), appShared.UseCaseInput[SearchUsersInput]{
		Base: appShared.BaseContext{Auth: &appShared.AuthContext{UserID: 501, Roles: []role.Code{role.User}}},
		Data: SearchUsersInput{PublicID: "hiro-public"},
	})

	require.ErrorIs(t, err, expectedErr)
}

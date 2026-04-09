package usecase

import (
	"context"
	"testing"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	domainaccount "github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	domainrole "github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	domainuser "github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type getProfileAccountRepoStub struct {
	findByID func(ctx context.Context, id shared.AccountID) (*domainaccount.Account, error)
}

func (s *getProfileAccountRepoStub) FindByID(ctx context.Context, id shared.AccountID) (*domainaccount.Account, error) {
	if s.findByID != nil {
		return s.findByID(ctx, id)
	}
	return nil, domainaccount.ErrAccountNotFound
}

func (*getProfileAccountRepoStub) FindByAccountName(context.Context, string) (*domainaccount.Account, error) {
	return nil, domainaccount.ErrAccountNotFound
}

func (*getProfileAccountRepoStub) FindByEmail(context.Context, shared.EmailAddress) (*domainaccount.Account, error) {
	return nil, domainaccount.ErrAccountNotFound
}

func (*getProfileAccountRepoStub) Create(context.Context, *domainaccount.Account) (shared.AccountID, error) {
	return 0, nil
}

func (*getProfileAccountRepoStub) Update(context.Context, *domainaccount.Account) error { return nil }

func (*getProfileAccountRepoStub) RegisterDevice(context.Context, *domainaccount.AccountDevice) error {
	return nil
}

func (*getProfileAccountRepoStub) RecordLoginEvent(context.Context, *domainaccount.AccountLoginEvent) error {
	return nil
}

func (*getProfileAccountRepoStub) ReplaceDevices(context.Context, shared.AccountID, []domainaccount.AccountDevice) error {
	return nil
}

type getProfileUserRepoStub struct {
	findByID func(ctx context.Context, id shared.UserID) (*domainuser.User, error)
}

func (*getProfileUserRepoStub) Create(context.Context, *domainuser.User) (shared.UserID, error) {
	return 0, nil
}

func (s *getProfileUserRepoStub) FindByID(ctx context.Context, id shared.UserID) (*domainuser.User, error) {
	if s.findByID != nil {
		return s.findByID(ctx, id)
	}
	return nil, domainuser.ErrUserNotFound
}

func (*getProfileUserRepoStub) FindByAccountID(context.Context, shared.AccountID) (*[]domainuser.User, error) {
	return nil, nil
}

func (*getProfileUserRepoStub) Update(context.Context, *domainuser.User) error { return nil }

func (*getProfileUserRepoStub) SearchByName(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}

func (*getProfileUserRepoStub) FindByAccountName(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}

func (*getProfileUserRepoStub) FindByPublicID(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}

func TestGetProfileUseCase_ReturnsProfile(t *testing.T) {
	email, err := shared.ParseEmail("profile@example.com")
	require.NoError(t, err)

	publicID, err := uuid.FromString("11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)

	accountRepo := &getProfileAccountRepoStub{
		findByID: func(context.Context, shared.AccountID) (*domainaccount.Account, error) {
			return &domainaccount.Account{
				ID:          shared.AccountID(101),
				PublicID:    publicID,
				Email:       email,
				AccountName: "profile_account",
				Status:      domainaccount.Active,
				UserIDs:     []shared.UserID{501, 502},
			}, nil
		},
	}
	userRepo := &getProfileUserRepoStub{
		findByID: func(context.Context, shared.UserID) (*domainuser.User, error) {
			return &domainuser.User{
				ID:        shared.UserID(501),
				Name:      "Profile User",
				Avatar:    "avatars/profile.png",
				RoleCodes: []domainrole.Code{domainrole.User},
			}, nil
		},
	}

	out, err := NewGetProfileUseCase(accountRepo, userRepo).Execute(context.Background(), appShared.UseCaseInput[GetProfileInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{
				AccountID: shared.AccountID(101),
				UserID:    shared.UserID(501),
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, int64(101), out.AccountID)
	assert.Equal(t, publicID.String(), out.PublicID)
	assert.Equal(t, "profile@example.com", out.Email)
	assert.Equal(t, "profile_account", out.AccountName)
	assert.Equal(t, "active", out.Status)
	assert.Equal(t, []int64{501, 502}, out.UserISs)
	assert.Equal(t, int64(501), out.CurrentUser.ID)
	assert.Equal(t, "Profile User", out.CurrentUser.Name)
	assert.Equal(t, "avatars/profile.png", out.CurrentUser.Avatar)
	assert.Equal(t, []string{"user"}, out.CurrentUser.RoleCodes)
}

func TestGetProfileUseCase_MapsMissingDependencies(t *testing.T) {
	t.Run("missing account", func(t *testing.T) {
		_, err := NewGetProfileUseCase(&getProfileAccountRepoStub{}, &getProfileUserRepoStub{}).Execute(context.Background(), appShared.UseCaseInput[GetProfileInput]{
			Base: appShared.BaseContext{
				Auth: &appShared.AuthContext{
					AccountID: shared.AccountID(101),
					UserID:    shared.UserID(501),
				},
			},
		})

		require.ErrorIs(t, err, ErrAccountNotFound)
	})

	t.Run("missing user", func(t *testing.T) {
		email, err := shared.ParseEmail("profile@example.com")
		require.NoError(t, err)

		_, err = NewGetProfileUseCase(&getProfileAccountRepoStub{
			findByID: func(context.Context, shared.AccountID) (*domainaccount.Account, error) {
				return &domainaccount.Account{
					ID:          shared.AccountID(101),
					PublicID:    uuid.Nil,
					Email:       email,
					AccountName: "profile_account",
					Status:      domainaccount.Active,
					UserIDs:     []shared.UserID{501},
				}, nil
			},
		}, &getProfileUserRepoStub{}).Execute(context.Background(), appShared.UseCaseInput[GetProfileInput]{
			Base: appShared.BaseContext{
				Auth: &appShared.AuthContext{
					AccountID: shared.AccountID(101),
					UserID:    shared.UserID(501),
				},
			},
		})

		require.ErrorIs(t, err, ErrUserNotFound)
	})
}

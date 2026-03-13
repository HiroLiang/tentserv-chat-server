package session

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/application/auth/port"
	"github.com/HiroLiang/goat-server/internal/domain/auth"
	"github.com/HiroLiang/goat-server/internal/domain/cache"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type SessionManager struct {
	cache cache.Cache
}

func NewSessionManager(cache cache.Cache) *SessionManager {
	return &SessionManager{cache: cache}
}

func (s SessionManager) Create(ctx context.Context, input auth.CreateSessionInput) (auth.TokenPair, error) {
	//TODO implement me
	panic("implement me")
}

func (s SessionManager) FindByToken(ctx context.Context, token auth.AccessToken) (*auth.Session, error) {
	//TODO implement me
	panic("implement me")
}

func (s SessionManager) Refresh(ctx context.Context, token auth.RefreshToken) (auth.TokenPair, error) {
	//TODO implement me
	panic("implement me")
}

func (s SessionManager) Revoke(ctx context.Context, token auth.AccessToken) error {
	//TODO implement me
	panic("implement me")
}

func (s SessionManager) RevokeAllForUser(ctx context.Context, userID shared.AccountID) error {
	//TODO implement me
	panic("implement me")
}

func (s SessionManager) RevokeAll(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (s SessionManager) SwitchUser(ctx context.Context, token auth.AccessToken, userID shared.UserID) error {
	//TODO implement me
	panic("implement me")
}

var _ port.SessionManager = (*SessionManager)(nil)

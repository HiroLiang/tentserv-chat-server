package mock

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/auth"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type MockSessionManager struct {
	CreateFn           func(ctx context.Context, input auth.CreateSessionInput) (auth.TokenPair, error)
	FindByTokenFn      func(ctx context.Context, token auth.AccessToken) (*auth.Session, error)
	RefreshFn          func(ctx context.Context, token auth.RefreshToken) (auth.TokenPair, error)
	RevokeFn           func(ctx context.Context, token auth.AccessToken) error
	RevokeAllForUserFn func(ctx context.Context, userID shared.AccountID) error
	RevokeAllFn        func(ctx context.Context) error
	SwitchUserFn       func(ctx context.Context, token auth.AccessToken, userID shared.UserID) error
}

var _ port.SessionManager = (*MockSessionManager)(nil)

func MockSessionManagerFactory() port.SessionManager {
	return &MockSessionManager{}
}

func (m *MockSessionManager) Create(ctx context.Context, input auth.CreateSessionInput) (auth.TokenPair, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, input)
	}
	return auth.TokenPair{}, nil
}

func (m *MockSessionManager) FindByToken(ctx context.Context, token auth.AccessToken) (*auth.Session, error) {
	if m.FindByTokenFn != nil {
		return m.FindByTokenFn(ctx, token)
	}
	return nil, auth.ErrSessionNotFound
}

func (m *MockSessionManager) Refresh(ctx context.Context, token auth.RefreshToken) (auth.TokenPair, error) {
	if m.RefreshFn != nil {
		return m.RefreshFn(ctx, token)
	}
	return auth.TokenPair{}, nil
}

func (m *MockSessionManager) Revoke(ctx context.Context, token auth.AccessToken) error {
	if m.RevokeFn != nil {
		return m.RevokeFn(ctx, token)
	}
	return nil
}

func (m *MockSessionManager) RevokeAllForUser(ctx context.Context, userID shared.AccountID) error {
	if m.RevokeAllForUserFn != nil {
		return m.RevokeAllForUserFn(ctx, userID)
	}
	return nil
}

func (m *MockSessionManager) RevokeAll(ctx context.Context) error {
	if m.RevokeAllFn != nil {
		return m.RevokeAllFn(ctx)
	}
	return nil
}

func (m *MockSessionManager) SwitchUser(ctx context.Context, token auth.AccessToken, userID shared.UserID) error {
	if m.SwitchUserFn != nil {
		return m.SwitchUserFn(ctx, token, userID)
	}
	return nil
}

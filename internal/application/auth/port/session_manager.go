package port

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/auth"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type SessionManager interface {
	Create(ctx context.Context, input auth.CreateSessionInput) (auth.TokenPair, error)
	FindByToken(ctx context.Context, token auth.AccessToken) (*auth.Session, error)
	Refresh(ctx context.Context, token auth.RefreshToken) (auth.TokenPair, error)
	Revoke(ctx context.Context, token auth.AccessToken) error
	RevokeAllForUser(ctx context.Context, userID shared.AccountID) error
	RevokeAll(ctx context.Context) error
	SwitchUser(ctx context.Context, token auth.AccessToken, userID shared.UserID) error
}

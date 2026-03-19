package auth

import (
	"context"

	session "github.com/HiroLiang/tentserv-chat-server/internal/domain/auth"
)

type TokenService interface {
	Generate(ctx context.Context, params session.CreateSessionInput) (string, error)
	Refresh(ctx context.Context, token string) (string, error)
	Validate(ctx context.Context, token string) (*session.Session, error)
	Revoke(ctx context.Context, token string) error
	RevokeAllForUser(ctx context.Context, userID string) error
	ListUserSessions(ctx context.Context, userID string) ([]*session.Session, error)
}

package mock

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/application/shared/auth"
	session "github.com/HiroLiang/goat-server/internal/domain/auth"
)

type MockTokenService struct{}

func MockTokenServiceFactory() auth.TokenService {
	return &MockTokenService{}
}

var _ auth.TokenService = (*MockTokenService)(nil)

func (m MockTokenService) Generate(ctx context.Context, params session.CreateSessionInput) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockTokenService) Refresh(ctx context.Context, token string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockTokenService) Validate(ctx context.Context, token string) (*session.Session, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockTokenService) Revoke(ctx context.Context, token string) error {
	//TODO implement me
	panic("implement me")
}

func (m MockTokenService) RevokeAllForUser(ctx context.Context, userID string) error {
	//TODO implement me
	panic("implement me")
}

func (m MockTokenService) ListUserSessions(ctx context.Context, userID string) ([]*session.Session, error) {
	//TODO implement me
	panic("implement me")
}

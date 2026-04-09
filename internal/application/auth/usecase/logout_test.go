package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/auth"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type logoutSessionManagerStub struct {
	revokeErr   error
	revokeCalls int
	lastToken   auth.AccessToken
}

func (s *logoutSessionManagerStub) Create(context.Context, auth.CreateSessionInput) (auth.TokenPair, error) {
	return auth.TokenPair{}, nil
}

func (s *logoutSessionManagerStub) FindByToken(context.Context, auth.AccessToken) (*auth.Session, error) {
	return nil, auth.ErrSessionNotFound
}

func (s *logoutSessionManagerStub) Refresh(context.Context, auth.RefreshToken) (auth.TokenPair, error) {
	return auth.TokenPair{}, nil
}

func (s *logoutSessionManagerStub) Revoke(_ context.Context, token auth.AccessToken) error {
	s.revokeCalls++
	s.lastToken = token
	return s.revokeErr
}

func (s *logoutSessionManagerStub) RevokeAllForUser(context.Context, shared.AccountID) error {
	return nil
}

func (s *logoutSessionManagerStub) RevokeAll(context.Context) error {
	return nil
}

func (s *logoutSessionManagerStub) SwitchUser(context.Context, auth.AccessToken, shared.UserID) error {
	return nil
}

func TestLogoutUseCase_RevokesCurrentAccessTokenHasStructuredLog(t *testing.T) {
	start := time.Now()
	sessionManager := &logoutSessionManagerStub{}
	uc := NewLogoutUseCase(sessionManager)
	input := &appShared.UseCaseInput[LogoutInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{
				AccessToken: auth.AccessToken("logout-token"),
			},
		},
	}

	t.Log("Given: authenticated logout request contains the current access token")
	t.Log("Input: token_present=true")
	t.Log("Action: execute logout usecase happy path")

	_, err := uc.Execute(context.Background(), input)

	t.Logf("Output: err=%v", err)
	t.Logf("Mutation: revoke_calls=%d revoked_token=%q", sessionManager.revokeCalls, sessionManager.lastToken)
	t.Logf("Duration: %s", time.Since(start))

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sessionManager.revokeCalls != 1 || sessionManager.lastToken != auth.AccessToken("logout-token") {
		t.Fatalf("expected revoke to receive current access token, got calls=%d token=%q", sessionManager.revokeCalls, sessionManager.lastToken)
	}
}

func TestLogoutUseCase_RevokeFailureMapsToLogoutFailedHasStructuredLog(t *testing.T) {
	start := time.Now()
	sessionManager := &logoutSessionManagerStub{revokeErr: errors.New("redis unavailable")}
	uc := NewLogoutUseCase(sessionManager)
	input := &appShared.UseCaseInput[LogoutInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{
				AccessToken: auth.AccessToken("logout-token"),
			},
		},
	}

	t.Log("Given: session manager revoke fails")
	t.Log("Input: token_present=true")
	t.Log("Action: execute logout usecase failure path")

	_, err := uc.Execute(context.Background(), input)

	t.Logf("Output: err=%v", err)
	t.Logf("Mutation: revoke_calls=%d revoked_token=%q", sessionManager.revokeCalls, sessionManager.lastToken)
	t.Logf("Duration: %s", time.Since(start))

	if !errors.Is(err, ErrLogoutFailed) {
		t.Fatalf("expected ErrLogoutFailed, got %v", err)
	}
	if sessionManager.revokeCalls != 1 {
		t.Fatalf("expected revoke to be attempted once, got %d", sessionManager.revokeCalls)
	}
}

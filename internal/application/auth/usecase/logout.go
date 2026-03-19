package usecase

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
)

type LogoutInput struct{}

type LogoutOutput struct{}

type LogoutUseCase struct {
	sessionManager port.SessionManager
}

func NewLogoutUseCase(sessionManager port.SessionManager) *LogoutUseCase {
	return &LogoutUseCase{
		sessionManager: sessionManager,
	}
}

func (uc *LogoutUseCase) Execute(
	ctx context.Context,
	input *appShared.UseCaseInput[LogoutInput],
) (LogoutOutput, error) {
	_ = uc.sessionManager.Revoke(ctx, input.Base.Auth.AccessToken)

	return LogoutOutput{}, nil
}

package usecase

import (
	"context"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type SwitchUserInput struct {
	TargetUserID shared.UserID
}

type SwitchUserUseCase struct {
	sessionManager port.SessionManager
}

func NewSwitchUserUseCase(sessionManager port.SessionManager) *SwitchUserUseCase {
	return &SwitchUserUseCase{sessionManager: sessionManager}
}

func (uc *SwitchUserUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[SwitchUserInput],
) error {
	if err := uc.sessionManager.SwitchUser(ctx, input.Base.Auth.AccessToken, input.Data.TargetUserID); err != nil {
		return ErrSwitchUser
	}
	return nil
}

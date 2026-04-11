package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type VerifyEmailInput struct {
	Token string
	Code  string
}

type VerifyEmailOutput struct{}

type VerifyEmailUseCase struct {
	verificationStore port.VerificationStore
	accountRepo       account.Repository
}

func NewVerifyEmailUseCase(
	verificationStore port.VerificationStore,
	accountRepo account.Repository,
) *VerifyEmailUseCase {
	return &VerifyEmailUseCase{
		verificationStore: verificationStore,
		accountRepo:       accountRepo,
	}
}

func (uc *VerifyEmailUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[VerifyEmailInput],
) (VerifyEmailOutput, error) {
	session, ok, err := uc.verificationStore.Get(ctx, input.Data.Token)
	if err != nil || !ok {
		return VerifyEmailOutput{}, ErrTokenInvalid
	}

	if verificationSessionTTL(session) <= 0 || session.ExpiresAtMS <= time.Now().UTC().UnixMilli() {
		_ = uc.verificationStore.Delete(ctx, input.Data.Token)
		return VerifyEmailOutput{}, ErrTokenInvalid
	}

	if session.Code != strings.TrimSpace(input.Data.Code) {
		remainingAttempts := session.RemainingAttempts - 1
		if remainingAttempts <= 0 {
			if err := uc.verificationStore.Delete(ctx, input.Data.Token); err != nil {
				return VerifyEmailOutput{}, ErrRegisterFailed
			}
			return VerifyEmailOutput{}, newVerificationAttemptError(ErrVerificationAttemptsExceeded, 0)
		}

		session.RemainingAttempts = remainingAttempts
		if err := uc.verificationStore.Store(ctx, input.Data.Token, session, verificationSessionTTL(session)); err != nil {
			return VerifyEmailOutput{}, ErrRegisterFailed
		}
		return VerifyEmailOutput{}, newVerificationAttemptError(ErrVerificationCodeInvalid, remainingAttempts)
	}

	if err := uc.verificationStore.Delete(ctx, input.Data.Token); err != nil {
		return VerifyEmailOutput{}, ErrRegisterFailed
	}

	acc, err := uc.accountRepo.FindByID(ctx, shared.AccountID(session.AccountID))
	if err != nil {
		return VerifyEmailOutput{}, ErrTokenInvalid
	}

	if acc.Status != account.Applying {
		return VerifyEmailOutput{}, ErrTokenInvalid
	}

	acc.SetStatus(account.Active)
	if err := uc.accountRepo.Update(ctx, acc); err != nil {
		return VerifyEmailOutput{}, ErrRegisterFailed
	}

	return VerifyEmailOutput{}, nil
}

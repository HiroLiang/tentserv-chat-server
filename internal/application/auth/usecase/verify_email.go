package usecase

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/application/auth/port"
	appShared "github.com/HiroLiang/goat-server/internal/application/shared"
	"github.com/HiroLiang/goat-server/internal/domain/account"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type VerifyEmailInput struct {
	Token string
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
	accountID, ok, err := uc.verificationStore.Get(ctx, input.Data.Token)
	if err != nil || !ok {
		return VerifyEmailOutput{}, ErrTokenInvalid
	}

	acc, err := uc.accountRepo.FindByID(ctx, shared.AccountID(accountID))
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

	_ = uc.verificationStore.Delete(ctx, input.Data.Token)

	return VerifyEmailOutput{}, nil
}

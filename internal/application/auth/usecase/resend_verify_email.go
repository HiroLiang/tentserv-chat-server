package usecase

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	appEmail "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type ResendVerifyEmailInput struct {
	Email string
}

type ResendVerifyEmailOutput struct{}

type ResendVerifyEmailUseCase struct {
	verificationStore  port.VerificationStore
	accountRepo        account.Repository
	emailService       appEmail.EmailService
	mailBuilderFactory func(recipientEmail, recipientName, verifyURL string) appEmail.EmailBuilder
}

func NewResendVerifyEmailUseCase(
	verificationStore port.VerificationStore,
	accountRepo account.Repository,
	emailService appEmail.EmailService,
	mailBuilderFactory func(recipientEmail, recipientName, verifyURL string) appEmail.EmailBuilder,
) *ResendVerifyEmailUseCase {
	return &ResendVerifyEmailUseCase{
		verificationStore:  verificationStore,
		accountRepo:        accountRepo,
		emailService:       emailService,
		mailBuilderFactory: mailBuilderFactory,
	}
}

func (uc *ResendVerifyEmailUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[ResendVerifyEmailInput],
) (ResendVerifyEmailOutput, error) {
	acc, err := uc.accountRepo.FindByEmail(ctx, shared.EmailAddress(input.Data.Email))
	if err != nil {
		return ResendVerifyEmailOutput{}, ErrRegisterFailed
	}

	if acc.Status != account.Applying {
		return ResendVerifyEmailOutput{}, ErrRegisterFailed
	}

	token, err := generateVerificationToken()
	if err != nil {
		return ResendVerifyEmailOutput{}, ErrRegisterFailed
	}

	conf := config.App()
	if err := uc.verificationStore.Store(ctx, token, int64(acc.ID), conf.Email.VerifyTTL); err != nil {
		return ResendVerifyEmailOutput{}, ErrRegisterFailed
	}

	verifyURL, err := buildVerificationURL(conf.Email.BaseURL, token)
	if err != nil {
		_ = uc.verificationStore.Delete(ctx, token)
		return ResendVerifyEmailOutput{}, ErrRegisterFailed
	}

	builder := uc.mailBuilderFactory(input.Data.Email, acc.AccountName, verifyURL)
	if err := uc.emailService.Send(ctx, builder); err != nil {
		_ = uc.verificationStore.Delete(ctx, token)
		return ResendVerifyEmailOutput{}, ErrRegisterFailed
	}

	return ResendVerifyEmailOutput{}, nil
}

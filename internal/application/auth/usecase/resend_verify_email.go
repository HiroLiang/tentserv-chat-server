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
	Token string
}

type ResendVerifyEmailOutput struct {
	VerificationToken       string
	VerificationExpiresAtMS int64
}

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
	currentSession, ok, err := uc.verificationStore.Get(ctx, input.Data.Token)
	if err != nil || !ok {
		return ResendVerifyEmailOutput{}, ErrTokenInvalid
	}

	acc, err := uc.accountRepo.FindByID(ctx, shared.AccountID(currentSession.AccountID))
	if err != nil {
		return ResendVerifyEmailOutput{}, ErrTokenInvalid
	}

	if acc.Status != account.Applying {
		return ResendVerifyEmailOutput{}, ErrTokenInvalid
	}

	if err := uc.verificationStore.Delete(ctx, input.Data.Token); err != nil {
		return ResendVerifyEmailOutput{}, ErrRegisterFailed
	}

	conf := config.App()
	token, session, err := newVerificationSession(int64(acc.ID), conf.Email.VerifyTTL)
	if err != nil {
		return ResendVerifyEmailOutput{}, ErrRegisterFailed
	}

	if err := uc.verificationStore.Store(ctx, token, session, conf.Email.VerifyTTL); err != nil {
		return ResendVerifyEmailOutput{}, ErrRegisterFailed
	}

	builder := uc.mailBuilderFactory(string(acc.Email), acc.AccountName, session.Code)
	if err := uc.emailService.Send(ctx, builder); err != nil {
		_ = uc.verificationStore.Delete(ctx, token)
		return ResendVerifyEmailOutput{}, ErrRegisterFailed
	}

	return ResendVerifyEmailOutput{
		VerificationToken:       token,
		VerificationExpiresAtMS: session.ExpiresAtMS,
	}, nil
}

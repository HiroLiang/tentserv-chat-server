package usecase

import (
	"context"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	appEmail "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type ResendLoginDeviceVerificationInput struct {
	Token string
}

type ResendLoginDeviceVerificationOutput struct {
	VerificationToken       string
	VerificationExpiresAtMS int64
}

type ResendLoginDeviceVerificationUseCase struct {
	verificationStore  port.VerificationStore
	accountRepo        account.Repository
	emailService       appEmail.EmailService
	mailBuilderFactory func(recipientEmail, recipientName, deviceName, deviceID, ip, verificationCode string, loginTime time.Time) appEmail.EmailBuilder
}

func NewResendLoginDeviceVerificationUseCase(
	verificationStore port.VerificationStore,
	accountRepo account.Repository,
	emailService appEmail.EmailService,
	mailBuilderFactory func(recipientEmail, recipientName, deviceName, deviceID, ip, verificationCode string, loginTime time.Time) appEmail.EmailBuilder,
) *ResendLoginDeviceVerificationUseCase {
	return &ResendLoginDeviceVerificationUseCase{
		verificationStore:  verificationStore,
		accountRepo:        accountRepo,
		emailService:       emailService,
		mailBuilderFactory: mailBuilderFactory,
	}
}

func (uc *ResendLoginDeviceVerificationUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[ResendLoginDeviceVerificationInput],
) (ResendLoginDeviceVerificationOutput, error) {
	currentSession, ok, err := uc.verificationStore.Get(ctx, input.Data.Token)
	if err != nil || !ok {
		return ResendLoginDeviceVerificationOutput{}, ErrTokenInvalid
	}

	acc, err := uc.accountRepo.FindByID(ctx, shared.AccountID(currentSession.AccountID))
	if err != nil || acc.Status != account.Active {
		return ResendLoginDeviceVerificationOutput{}, ErrTokenInvalid
	}

	if err := uc.verificationStore.Delete(ctx, input.Data.Token); err != nil {
		return ResendLoginDeviceVerificationOutput{}, ErrLoginFailed
	}

	conf := config.App()
	token, session, err := newVerificationSession(int64(acc.ID), conf.Email.VerifyTTL)
	if err != nil {
		return ResendLoginDeviceVerificationOutput{}, ErrLoginFailed
	}
	session.DeviceID = currentSession.DeviceID
	session.DeviceName = currentSession.DeviceName
	session.IP = currentSession.IP
	session.UserAgent = currentSession.UserAgent

	if err := uc.verificationStore.Store(ctx, token, session, conf.Email.VerifyTTL); err != nil {
		return ResendLoginDeviceVerificationOutput{}, ErrLoginFailed
	}

	builder := uc.mailBuilderFactory(
		string(acc.Email),
		acc.AccountName,
		session.DeviceName,
		session.DeviceID,
		session.IP,
		session.Code,
		time.Now(),
	)
	if err := uc.emailService.Send(ctx, builder); err != nil {
		_ = uc.verificationStore.Delete(ctx, token)
		return ResendLoginDeviceVerificationOutput{}, ErrLoginFailed
	}

	return ResendLoginDeviceVerificationOutput{
		VerificationToken:       token,
		VerificationExpiresAtMS: session.ExpiresAtMS,
	}, nil
}

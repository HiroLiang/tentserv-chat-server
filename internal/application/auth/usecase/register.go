package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	appEmail "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared/security"
	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/gofrs/uuid"
)

type RegisterInput struct {
	Name     string
	Account  string
	Email    string
	Password string
}

type RegisterOutput struct {
	ID int64
}

type RegisterUseCase struct {
	uow                transaction.UnitOfWork
	hasher             security.Hasher
	accountRepo        account.Repository
	userRepo           user.Repository
	verificationStore  port.VerificationStore
	emailService       appEmail.EmailService
	mailBuilderFactory func(recipientEmail, recipientName, verifyURL string) appEmail.EmailBuilder
}

func NewRegisterUseCase(
	uow transaction.UnitOfWork,
	hasher security.Hasher,
	accountRepo account.Repository,
	userRepo user.Repository,
	verificationStore port.VerificationStore,
	emailService appEmail.EmailService,
	mailBuilderFactory func(recipientEmail, recipientName, verifyURL string) appEmail.EmailBuilder,
) *RegisterUseCase {
	return &RegisterUseCase{
		uow:                uow,
		hasher:             hasher,
		accountRepo:        accountRepo,
		userRepo:           userRepo,
		verificationStore:  verificationStore,
		emailService:       emailService,
		mailBuilderFactory: mailBuilderFactory,
	}
}

func (uc *RegisterUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[RegisterInput],
) (RegisterOutput, error) {
	ctx, tx, err := uc.uow.Begin(ctx)
	if err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Validate email
	emailAddr, err := shared.ParseEmail(input.Data.Email)
	if err != nil {
		return RegisterOutput{}, ErrInvalidEmail
	}

	// Hash password
	hash, err := uc.hasher.Hash(input.Data.Password)
	if err != nil {
		return RegisterOutput{}, ErrInvalidPassword
	}

	// Generate public ID
	publicID, err := uuid.NewV4()
	if err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}

	// Create Account
	newAccount := account.NewAccount(
		publicID,
		emailAddr,
		input.Data.Account,
		hash,
		1,
	)
	accountId, err := uc.accountRepo.Create(ctx, newAccount)
	if err != nil {
		switch {
		case errors.Is(err, account.ErrAccountExist):
			return RegisterOutput{}, ErrAccountExist
		case errors.Is(err, account.ErrEmailExist):
			return RegisterOutput{}, ErrEmailExist
		default:
			return RegisterOutput{}, ErrRegisterFailed
		}
	}

	// Generate verification token
	token, err := generateVerificationToken()
	if err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}

	// Store token → accountID in Redis
	conf := config.App()
	if err := uc.verificationStore.Store(ctx, token, int64(accountId), conf.Email.VerifyTTL); err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}

	// Build, verify URL and send email
	verifyURL := fmt.Sprintf("%s/api/auth/verify-email?token=%s", conf.Email.BaseURL, token)
	builder := uc.mailBuilderFactory(input.Data.Email, input.Data.Name, verifyURL)
	if err := uc.emailService.Send(ctx, builder); err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}

	return RegisterOutput{int64(accountId)}, tx.Commit()
}

func generateVerificationToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

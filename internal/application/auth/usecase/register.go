package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	appEmail "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared/security"
	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userrole"
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
	userRoleRepo       userrole.Repository
	verificationStore  port.VerificationStore
	emailService       appEmail.EmailService
	mailBuilderFactory func(recipientEmail, recipientName, verifyURL string) appEmail.EmailBuilder
}

func NewRegisterUseCase(
	uow transaction.UnitOfWork,
	hasher security.Hasher,
	accountRepo account.Repository,
	userRepo user.Repository,
	userRoleRepo userrole.Repository,
	verificationStore port.VerificationStore,
	emailService appEmail.EmailService,
	mailBuilderFactory func(recipientEmail, recipientName, verifyURL string) appEmail.EmailBuilder,
) *RegisterUseCase {
	return &RegisterUseCase{
		uow:                uow,
		hasher:             hasher,
		accountRepo:        accountRepo,
		userRepo:           userRepo,
		userRoleRepo:       userRoleRepo,
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
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Reject emails that exceed RFC 5321 maximum length
	if len(input.Data.Email) > 254 {
		return RegisterOutput{}, ErrInvalidEmail
	}

	// Validate email
	emailAddr, err := shared.ParseEmail(input.Data.Email)
	if err != nil {
		return RegisterOutput{}, ErrInvalidEmail
	}

	if _, err := uc.accountRepo.FindByEmail(ctx, emailAddr); err == nil {
		return RegisterOutput{}, ErrEmailExist
	} else if !errors.Is(err, account.ErrAccountNotFound) {
		return RegisterOutput{}, ErrRegisterFailed
	}

	// Validate account name format (alphanumeric + underscore only)
	if !accountNameRegex.MatchString(input.Data.Account) {
		return RegisterOutput{}, ErrInvalidAccount
	}

	if _, err := uc.accountRepo.FindByAccountName(ctx, input.Data.Account); err == nil {
		return RegisterOutput{}, ErrAccountExist
	} else if !errors.Is(err, account.ErrAccountNotFound) {
		return RegisterOutput{}, ErrRegisterFailed
	}

	// Reject common/weak passwords
	if _, weak := commonPasswords[strings.ToLower(input.Data.Password)]; weak {
		return RegisterOutput{}, ErrWeakPassword
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
	newAccount.ID = accountId

	// Create a user with the provided display name
	newUser := user.NewUser(accountId, input.Data.Name)
	userID, err := uc.userRepo.Create(ctx, newUser)
	if err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}

	// Assign default roles
	for _, code := range newUser.RoleCodes {
		if err := uc.userRoleRepo.Assign(ctx, userID, code); err != nil {
			return RegisterOutput{}, ErrRegisterFailed
		}
	}

	// Link user to account
	newAccount.AddUser(userID)
	if err := uc.accountRepo.Update(ctx, newAccount); err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}

	if err := tx.Commit(); err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}
	committed = true

	// Generate verification token
	token, err := generateVerificationToken()
	if err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}

	conf := config.App()
	if err := uc.verificationStore.Store(ctx, token, int64(accountId), conf.Email.VerifyTTL); err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}

	verifyURL, err := buildVerificationURL(conf.Email.BaseURL, token)
	if err != nil {
		_ = uc.verificationStore.Delete(ctx, token)
		return RegisterOutput{}, ErrRegisterFailed
	}

	builder := uc.mailBuilderFactory(input.Data.Email, input.Data.Name, verifyURL)
	if err := uc.emailService.Send(ctx, builder); err != nil {
		_ = uc.verificationStore.Delete(ctx, token)
		return RegisterOutput{}, ErrRegisterFailed
	}

	return RegisterOutput{int64(accountId)}, nil
}

var accountNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

var commonPasswords = map[string]struct{}{
	"123456": {}, "password": {}, "12345678": {}, "qwerty": {},
	"123456789": {}, "12345": {}, "1234567": {}, "password1": {},
	"iloveyou": {}, "abc123": {}, "1234567890": {}, "123123": {},
	"111111": {}, "letmein": {}, "monkey": {}, "dragon": {},
	"master": {}, "sunshine": {}, "princess": {}, "welcome": {},
	"shadow": {}, "batman": {}, "football": {}, "1q2w3e4r": {},
}

func generateVerificationToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func buildVerificationURL(baseURL, token string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/auth/verify-email"
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

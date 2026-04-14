package usecase

import (
	"context"
	"errors"
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
	Name            string
	Account         string
	Email           string
	Password        string
	ConfirmPassword string
}

type RegisterOutput struct {
	ID                      int64
	VerificationToken       string
	VerificationExpiresAtMS int64
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

	if input.Data.Password != input.Data.ConfirmPassword {
		return RegisterOutput{}, ErrPasswordConfirmMismatch
	}

	// Validate email
	emailAddr, err := shared.ParseEmail(input.Data.Email)
	if err != nil {
		return RegisterOutput{}, ErrInvalidEmail
	}

	existingApplyingAccount, err := uc.findApplyingAccountByEmail(ctx, emailAddr)
	if err != nil {
		if errors.Is(err, ErrVerificationPending) || errors.Is(err, ErrEmailExist) {
			return RegisterOutput{}, err
		}
		return RegisterOutput{}, ErrRegisterFailed
	}

	// Validate account name format (alphanumeric + underscore only)
	if !accountNameRegex.MatchString(input.Data.Account) {
		return RegisterOutput{}, ErrInvalidAccount
	}

	if err := uc.ensureAccountNameAvailable(ctx, input.Data.Account, existingApplyingAccount); err != nil {
		if errors.Is(err, ErrAccountExist) {
			return RegisterOutput{}, err
		}
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

	accountID, recipientName, err := uc.upsertApplyingAccount(ctx, existingApplyingAccount, emailAddr, input, hash)
	if err != nil {
		if errors.Is(err, ErrAccountExist) || errors.Is(err, ErrEmailExist) {
			return RegisterOutput{}, err
		}
		return RegisterOutput{}, ErrRegisterFailed
	}

	if err := tx.Commit(); err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}
	committed = true

	conf := config.App()
	token, session, err := newVerificationSession(int64(accountID), conf.Email.VerifyTTL)
	if err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}

	if err := uc.verificationStore.Store(ctx, token, session, conf.Email.VerifyTTL); err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}

	builder := uc.mailBuilderFactory(input.Data.Email, recipientName, session.Code)
	if err := uc.emailService.Send(ctx, builder); err != nil {
		_ = uc.verificationStore.Delete(ctx, token)
		return RegisterOutput{}, ErrRegisterFailed
	}

	return RegisterOutput{
		ID:                      int64(accountID),
		VerificationToken:       token,
		VerificationExpiresAtMS: session.ExpiresAtMS,
	}, nil
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

func (uc *RegisterUseCase) findApplyingAccountByEmail(ctx context.Context, email shared.EmailAddress) (*account.Account, error) {
	acc, err := uc.accountRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, account.ErrAccountNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if acc.Status != account.Applying {
		return nil, ErrEmailExist
	}

	if _, ok, err := uc.verificationStore.FindTokenByAccountID(ctx, int64(acc.ID)); err != nil {
		return nil, err
	} else if ok {
		return nil, ErrVerificationPending
	}

	return acc, nil
}

func (uc *RegisterUseCase) ensureAccountNameAvailable(ctx context.Context, accountName string, existingApplyingAccount *account.Account) error {
	acc, err := uc.accountRepo.FindByAccountName(ctx, accountName)
	if err != nil {
		if errors.Is(err, account.ErrAccountNotFound) {
			return nil
		}
		return err
	}

	if existingApplyingAccount != nil && acc.ID == existingApplyingAccount.ID {
		return nil
	}

	return ErrAccountExist
}

func (uc *RegisterUseCase) upsertApplyingAccount(
	ctx context.Context,
	existingApplyingAccount *account.Account,
	email shared.EmailAddress,
	input appShared.UseCaseInput[RegisterInput],
	passwordHash string,
) (shared.AccountID, string, error) {
	if existingApplyingAccount == nil {
		return uc.createApplyingAccount(ctx, email, input, passwordHash)
	}
	return uc.updateApplyingAccount(ctx, existingApplyingAccount, email, input, passwordHash)
}

func (uc *RegisterUseCase) createApplyingAccount(
	ctx context.Context,
	email shared.EmailAddress,
	input appShared.UseCaseInput[RegisterInput],
	passwordHash string,
) (shared.AccountID, string, error) {
	publicID, err := uuid.NewV4()
	if err != nil {
		return 0, "", err
	}

	newAccount := account.NewAccount(
		publicID,
		email,
		input.Data.Account,
		passwordHash,
		1,
	)
	accountID, err := uc.accountRepo.Create(ctx, newAccount)
	if err != nil {
		switch {
		case errors.Is(err, account.ErrAccountExist):
			return 0, "", ErrAccountExist
		case errors.Is(err, account.ErrEmailExist):
			return 0, "", ErrEmailExist
		default:
			return 0, "", err
		}
	}
	newAccount.ID = accountID

	newUser := user.NewUser(accountID, input.Data.Name)
	userID, err := uc.userRepo.Create(ctx, newUser)
	if err != nil {
		return 0, "", err
	}

	for _, code := range newUser.RoleCodes {
		if err := uc.userRoleRepo.Assign(ctx, userID, code); err != nil {
			return 0, "", err
		}
	}

	newAccount.AddUser(userID)
	if err := uc.accountRepo.Update(ctx, newAccount); err != nil {
		return 0, "", err
	}

	return accountID, input.Data.Name, nil
}

func (uc *RegisterUseCase) updateApplyingAccount(
	ctx context.Context,
	acc *account.Account,
	email shared.EmailAddress,
	input appShared.UseCaseInput[RegisterInput],
	passwordHash string,
) (shared.AccountID, string, error) {
	acc.Email = email
	acc.AccountName = input.Data.Account
	acc.Password = passwordHash

	if err := uc.accountRepo.Update(ctx, acc); err != nil {
		return 0, "", err
	}

	accountUsers, err := uc.userRepo.FindByAccountID(ctx, acc.ID)
	if err != nil {
		return 0, "", err
	}
	if accountUsers == nil || len(*accountUsers) == 0 {
		return 0, "", user.ErrUserNotFound
	}

	primaryUser := (*accountUsers)[0]
	primaryUser.Name = input.Data.Name
	if err := uc.userRepo.Update(ctx, &primaryUser); err != nil {
		return 0, "", err
	}

	return acc.ID, primaryUser.Name, nil
}

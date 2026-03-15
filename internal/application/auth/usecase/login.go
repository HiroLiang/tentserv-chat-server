package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/HiroLiang/goat-server/internal/application/auth/port"
	appShared "github.com/HiroLiang/goat-server/internal/application/shared"
	appEmail "github.com/HiroLiang/goat-server/internal/application/shared/email"
	"github.com/HiroLiang/goat-server/internal/application/shared/security"
	"github.com/HiroLiang/goat-server/internal/domain/account"
	"github.com/HiroLiang/goat-server/internal/domain/auth"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
	"github.com/HiroLiang/goat-server/internal/domain/transaction"
	"github.com/HiroLiang/goat-server/internal/domain/user"
)

type LoginInput struct {
	Identifier string
	Password   string
	DeviceID   string
}

type LoginOutput struct {
	TokenPair auth.TokenPair
}

type LoginUseCase struct {
	uow                transaction.UnitOfWork
	hasher             security.Hasher
	sessionManager     port.SessionManager
	accountRepo        account.Repository
	userRepo           user.Repository
	emailService       appEmail.EmailService
	mailBuilderFactory func(recipientEmail, recipientName, deviceID, ip string, loginTime time.Time) appEmail.EmailBuilder
}

func NewLoginUseCase(
	uow transaction.UnitOfWork,
	hasher security.Hasher,
	sessionManager port.SessionManager,
	accountRepo account.Repository,
	userRepo user.Repository,
	emailService appEmail.EmailService,
	mailBuilderFactory func(recipientEmail, recipientName, deviceID, ip string, loginTime time.Time) appEmail.EmailBuilder,
) *LoginUseCase {
	return &LoginUseCase{
		uow:                uow,
		hasher:             hasher,
		sessionManager:     sessionManager,
		accountRepo:        accountRepo,
		userRepo:           userRepo,
		emailService:       emailService,
		mailBuilderFactory: mailBuilderFactory,
	}
}

func (uc *LoginUseCase) Execute(ctx context.Context, input *appShared.UseCaseInput[LoginInput]) (LoginOutput, error) {

	// Begin transaction
	ctx, tx, err := uc.uow.Begin(ctx)
	if err != nil {
		return LoginOutput{}, ErrLoginFailed
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Find an account by identifier (email or account name)
	accountData, err := uc.findAccount(ctx, input.Data.Identifier)
	if err != nil {
		if errors.Is(err, account.ErrAccountNotFound) {
			return LoginOutput{}, ErrAccountNotFound
		}
		return LoginOutput{}, ErrLoginFailed
	}

	// Check account status
	if err := uc.castStatusError(accountData.Status); err != nil {
		return LoginOutput{}, err
	}

	// Verify password
	if !uc.hasher.Verify(input.Data.Password, accountData.Password) {
		return LoginOutput{}, ErrPasswordError
	}

	// Create a user if not exists
	var primaryUserID shared.UserID

	if len(accountData.UserIDs) == 0 {
		newUser := user.NewUser(accountData.ID, accountData.AccountName)
		userID, err := uc.userRepo.Create(ctx, newUser)
		if err != nil {
			return LoginOutput{}, ErrLoginFailed
		}

		// add user to account
		accountData.AddUser(userID)

		primaryUserID = userID
	} else {
		primaryUserID = accountData.UserIDs[0]
	}

	// Parse device ID
	deviceID, err := shared.ParseDeviceID(input.Data.DeviceID)
	if err != nil {
		return LoginOutput{}, ErrInvalidDeviceID
	}

	// Update device
	device := account.AccountDevice{
		AccountID:  accountData.ID,
		DeviceID:   deviceID,
		LastIP:     input.Base.Request.IP,
		LastSeenAt: time.Now(),
	}

	err = uc.accountRepo.RegisterDevice(ctx, &device)
	if err != nil {
		return LoginOutput{}, ErrLoginFailed
	}

	// Create session
	tokenPair, err := uc.sessionManager.Create(ctx, auth.CreateSessionInput{
		AccountID: accountData.ID,
		UserID:    primaryUserID,
		DeviceID:  deviceID,
	})
	if err != nil {
		return LoginOutput{}, ErrLoginFailed
	}

	// Update account
	err = uc.accountRepo.Update(ctx, accountData)
	if err != nil {
		return LoginOutput{}, ErrLoginFailed
	}

	// Send login notification email (fire-and-forget)
	go func() {
		bgCtx := context.Background()
		builder := uc.mailBuilderFactory(
			string(accountData.Email),
			accountData.AccountName,
			input.Data.DeviceID,
			input.Base.Request.IP.String(),
			time.Now(),
		)
		_ = uc.emailService.Send(bgCtx, builder)
	}()

	return LoginOutput{TokenPair: tokenPair}, tx.Commit()
}

func (uc *LoginUseCase) findAccount(ctx context.Context, identifier string) (*account.Account, error) {
	if emailAddr, err := shared.ParseEmail(identifier); err == nil {
		acc, err := uc.accountRepo.FindByEmail(ctx, emailAddr)
		if err == nil {
			return acc, nil
		}
		if !errors.Is(err, account.ErrAccountNotFound) {
			return nil, err
		}
	}
	return uc.accountRepo.FindByAccountName(ctx, identifier)
}

func (uc *LoginUseCase) castStatusError(status account.Status) error {
	switch status {
	case account.Active:
		return nil
	case account.Banned:
		return ErrAccountBanned
	case account.Applying:
		return ErrAccountApplying
	case account.Inactive:
		return ErrAccountInactive
	case account.Deleted:
		return ErrAccountNotFound
	default:
		return ErrLoginFailed
	}
}

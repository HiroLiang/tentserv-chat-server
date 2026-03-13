package usecase

import (
	"context"
	"errors"

	"github.com/HiroLiang/goat-server/internal/application/auth/port"
	appShared "github.com/HiroLiang/goat-server/internal/application/shared"
	"github.com/HiroLiang/goat-server/internal/application/shared/security"
	"github.com/HiroLiang/goat-server/internal/domain/account"
	"github.com/HiroLiang/goat-server/internal/domain/auth"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
	"github.com/HiroLiang/goat-server/internal/domain/transaction"
	"github.com/HiroLiang/goat-server/internal/domain/user"
)

type LoginInput struct {
	AccountName string
	Password    string
	DeviceID    string
}

type LoginOutput struct {
	TokenPair auth.TokenPair
}

type LoginUseCase struct {
	uow            transaction.UnitOfWork
	hasher         security.Hasher
	sessionManager port.SessionManager
	accountRepo    account.Repository
	userRepo       user.Repository
}

func NewLoginUseCase(
	uow transaction.UnitOfWork,
	hasher security.Hasher,
	sessionManager port.SessionManager,
	accountRepo account.Repository,
	userRepo user.Repository,
) *LoginUseCase {
	return &LoginUseCase{
		uow:            uow,
		hasher:         hasher,
		sessionManager: sessionManager,
		accountRepo:    accountRepo,
		userRepo:       userRepo,
	}
}

func (uc *LoginUseCase) Execute(ctx context.Context, input *appShared.UseCaseInput[LoginInput]) (LoginOutput, error) {

	// Begin transaction
	ctx, tx, err := uc.uow.Begin(ctx)
	if err != nil {
		return LoginOutput{}, ErrLoginFailed
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Find account
	accountData, err := uc.accountRepo.FindByAccountName(ctx, input.Data.AccountName)
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

	if !accountData.HasDevice(deviceID) {
		accountData.AddDevice(deviceID)
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

	// TODO: Record login event & send email notify login device

	return LoginOutput{TokenPair: tokenPair}, tx.Commit()
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

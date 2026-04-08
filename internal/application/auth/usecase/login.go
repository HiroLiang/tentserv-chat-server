package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	appEmail "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared/security"
	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/auth"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userrole"
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
	userRoleRepo       userrole.Repository
	deviceRepo         device.Repository
	participantRepo    participant.Repository
	emailService       appEmail.EmailService
	mailBuilderFactory func(recipientEmail, recipientName, deviceName, deviceID, ip string, loginTime time.Time) appEmail.EmailBuilder
}

func NewLoginUseCase(
	uow transaction.UnitOfWork,
	hasher security.Hasher,
	sessionManager port.SessionManager,
	accountRepo account.Repository,
	userRepo user.Repository,
	userRoleRepo userrole.Repository,
	deviceRepo device.Repository,
	participantRepo participant.Repository,
	emailService appEmail.EmailService,
	mailBuilderFactory func(recipientEmail, recipientName, deviceName, deviceID, ip string, loginTime time.Time) appEmail.EmailBuilder,
) *LoginUseCase {
	return &LoginUseCase{
		uow:                uow,
		hasher:             hasher,
		sessionManager:     sessionManager,
		accountRepo:        accountRepo,
		userRepo:           userRepo,
		userRoleRepo:       userRoleRepo,
		deviceRepo:         deviceRepo,
		participantRepo:    participantRepo,
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
	committed := false
	defer func() {
		if !committed {
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

	// Parse device ID and require that the device was registered during startup.
	deviceID, err := shared.ParseDeviceID(input.Data.DeviceID)
	if err != nil {
		return LoginOutput{}, ErrInvalidDeviceID
	}
	deviceData, err := uc.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			return LoginOutput{}, ErrInvalidDeviceID
		}
		return LoginOutput{}, ErrLoginFailed
	}

	// Create a user if not exists
	var primaryUserID shared.UserID

	if len(accountData.UserIDs) == 0 {
		newUser := user.NewUser(accountData.ID, accountData.AccountName)
		userID, err := uc.userRepo.Create(ctx, newUser)
		if err != nil {
			return LoginOutput{}, ErrLoginFailed
		}

		// persist default roles
		for _, code := range newUser.RoleCodes {
			if err := uc.userRoleRepo.Assign(ctx, userID, code); err != nil {
				return LoginOutput{}, ErrLoginFailed
			}
		}

		// add user to account
		accountData.AddUser(userID)

		primaryUserID = userID
	} else {
		primaryUserID = accountData.UserIDs[0]
	}

	if err := uc.ensureParticipant(ctx, primaryUserID); err != nil {
		return LoginOutput{}, ErrLoginFailed
	}

	// Update device
	accountDevice := account.AccountDevice{
		AccountID:  accountData.ID,
		DeviceID:   deviceID,
		LastIP:     input.Base.Request.IP,
		LastSeenAt: time.Now(),
	}

	err = uc.accountRepo.RegisterDevice(ctx, &accountDevice)
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

	err = uc.accountRepo.RecordLoginEvent(ctx, &account.AccountLoginEvent{
		AccountID: accountData.ID,
		DeviceID:  deviceID,
		IPAddress: input.Base.Request.IP,
		UserAgent: input.Base.Request.UserAgent,
		Success:   true,
	})
	if err != nil {
		return LoginOutput{}, ErrLoginFailed
	}

	// Update account
	err = uc.accountRepo.Update(ctx, accountData)
	if err != nil {
		return LoginOutput{}, ErrLoginFailed
	}

	err = tx.Commit()
	if err != nil {
		return LoginOutput{}, ErrLoginFailed
	}
	committed = true

	// Send login notification email (fire-and-forget)
	if config.Env("APP_ENV", "dev") != "dev" {
		go func() {
			bgCtx := context.Background()
			builder := uc.mailBuilderFactory(
				string(accountData.Email),
				accountData.AccountName,
				deviceData.Name,
				input.Data.DeviceID,
				input.Base.Request.IP.String(),
				time.Now(),
			)
			_ = uc.emailService.Send(bgCtx, builder)
		}()
	}

	return LoginOutput{TokenPair: tokenPair}, nil
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

func (uc *LoginUseCase) ensureParticipant(ctx context.Context, userID shared.UserID) error {
	_, err := uc.participantRepo.FindByUserID(ctx, userID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, participant.ErrNotFound) {
		return err
	}

	p := participant.Participant{
		Type:   participant.UserType,
		UserID: &userID,
	}
	if err := uc.participantRepo.Create(ctx, &p); err != nil {
		if errors.Is(err, participant.ErrAlreadyExists) {
			return nil
		}
		return err
	}
	return nil
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

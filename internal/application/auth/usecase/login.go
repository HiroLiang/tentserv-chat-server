package usecase

import (
	"context"
	"errors"
	"strings"
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
	domainSecurity "github.com/HiroLiang/tentserv-chat-server/internal/domain/security"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
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
	LoginStatus             string
	TokenPair               auth.TokenPair
	VerificationToken       string
	VerificationExpiresAtMS int64
}

type LoginUseCase struct {
	uow                            transaction.UnitOfWork
	hasher                         security.Hasher
	loginLimiter                   security.LoginRateLimiter
	sessionManager                 port.SessionManager
	verificationStore              port.VerificationStore
	accountRepo                    account.Repository
	userRepo                       user.Repository
	userRoleRepo                   userrole.Repository
	deviceRepo                     device.Repository
	participantRepo                participant.Repository
	selfSyncRepo                   selfsenderkeysync.Repository
	emailService                   appEmail.EmailService
	loginMailBuilderFactory        func(recipientEmail, recipientName, deviceName, deviceID, ip string, loginTime time.Time) appEmail.EmailBuilder
	verificationMailBuilderFactory func(recipientEmail, recipientName, deviceName, deviceID, ip, verificationCode string, loginTime time.Time) appEmail.EmailBuilder
}

func NewLoginUseCase(
	uow transaction.UnitOfWork,
	hasher security.Hasher,
	loginLimiter security.LoginRateLimiter,
	sessionManager port.SessionManager,
	verificationStore port.VerificationStore,
	accountRepo account.Repository,
	userRepo user.Repository,
	userRoleRepo userrole.Repository,
	deviceRepo device.Repository,
	participantRepo participant.Repository,
	selfSyncRepo selfsenderkeysync.Repository,
	emailService appEmail.EmailService,
	loginMailBuilderFactory func(recipientEmail, recipientName, deviceName, deviceID, ip string, loginTime time.Time) appEmail.EmailBuilder,
	verificationMailBuilderFactory func(recipientEmail, recipientName, deviceName, deviceID, ip, verificationCode string, loginTime time.Time) appEmail.EmailBuilder,
) *LoginUseCase {
	return &LoginUseCase{
		uow:                            uow,
		hasher:                         hasher,
		loginLimiter:                   loginLimiter,
		sessionManager:                 sessionManager,
		verificationStore:              verificationStore,
		accountRepo:                    accountRepo,
		userRepo:                       userRepo,
		userRoleRepo:                   userRoleRepo,
		deviceRepo:                     deviceRepo,
		participantRepo:                participantRepo,
		selfSyncRepo:                   selfSyncRepo,
		emailService:                   emailService,
		loginMailBuilderFactory:        loginMailBuilderFactory,
		verificationMailBuilderFactory: verificationMailBuilderFactory,
	}
}

func (uc *LoginUseCase) Execute(ctx context.Context, input *appShared.UseCaseInput[LoginInput]) (LoginOutput, error) {
	normalizedIdentifier := normalizeLoginIdentifier(input.Data.Identifier)

	if err := uc.checkLoginAttempt(ctx, input.Base.Request.IP.String(), normalizedIdentifier); err != nil {
		return LoginOutput{}, err
	}

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
	accountData, err := uc.findAccount(ctx, normalizedIdentifier)
	if err != nil {
		if errors.Is(err, account.ErrAccountNotFound) {
			uc.recordFailedLoginAttempt(ctx, input.Base.Request.IP.String(), normalizedIdentifier)
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
		uc.recordFailedLoginAttempt(ctx, input.Base.Request.IP.String(), normalizedIdentifier)
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

	existingDevice := accountData.GetDevice(deviceID)
	readyOtherDevices := accountData.CountReadyDevicesExcluding(deviceID)
	if readyOtherDevices > 0 {
		if err := uc.ensureNoActiveSelfSync(ctx, accountData, deviceID); err != nil {
			return LoginOutput{}, err
		}
	}

	if uc.requiresDeviceVerification(existingDevice, readyOtherDevices) {
		if err := uc.accountRepo.RegisterDevice(ctx, &account.AccountDevice{
			AccountID:  accountData.ID,
			DeviceID:   deviceID,
			Status:     account.DeviceStatusPendingVerification,
			LastIP:     input.Base.Request.IP,
			LastSeenAt: time.Now(),
		}); err != nil {
			return LoginOutput{}, ErrLoginFailed
		}
		if err := tx.Commit(); err != nil {
			return LoginOutput{}, ErrLoginFailed
		}
		committed = true

		conf := config.App()
		token, session, err := newVerificationSession(int64(accountData.ID), conf.Email.VerifyTTL)
		if err != nil {
			return LoginOutput{}, ErrLoginFailed
		}
		session.DeviceID = deviceID.String()
		session.DeviceName = deviceData.Name
		session.IP = input.Base.Request.IP.String()
		session.UserAgent = input.Base.Request.UserAgent

		if err := uc.verificationStore.Store(ctx, token, session, conf.Email.VerifyTTL); err != nil {
			return LoginOutput{}, ErrLoginFailed
		}

		builder := uc.verificationMailBuilderFactory(
			string(accountData.Email),
			accountData.AccountName,
			deviceData.Name,
			deviceID.String(),
			input.Base.Request.IP.String(),
			session.Code,
			time.Now(),
		)
		if err := uc.emailService.Send(ctx, builder); err != nil {
			_ = uc.verificationStore.Delete(ctx, token)
			return LoginOutput{}, ErrLoginFailed
		}

		return LoginOutput{
			LoginStatus:             "device_verification_required",
			VerificationToken:       token,
			VerificationExpiresAtMS: session.ExpiresAtMS,
		}, nil
	}

	// Create a user if not exists
	primaryUserID, err := uc.ensurePrimaryUser(ctx, accountData)
	if err != nil {
		return LoginOutput{}, ErrLoginFailed
	}

	if err := uc.ensureParticipant(ctx, primaryUserID); err != nil {
		return LoginOutput{}, ErrLoginFailed
	}

	// Update device
	accountDevice := account.AccountDevice{
		AccountID:  accountData.ID,
		DeviceID:   deviceID,
		Status:     uc.directLoginDeviceStatus(existingDevice),
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
	uc.recordSuccessfulLoginAttempt(ctx, input.Base.Request.IP.String(), normalizedIdentifier)

	// Send login notification email (fire-and-forget)
	if config.Env("APP_ENV", "dev") != "dev" {
		go func() {
			bgCtx := context.Background()
			builder := uc.loginMailBuilderFactory(
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

	return LoginOutput{
		LoginStatus: "authenticated",
		TokenPair:   tokenPair,
	}, nil
}

func (uc *LoginUseCase) findAccount(ctx context.Context, identifier string) (*account.Account, error) {
	if emailAddr, err := shared.ParseEmail(identifier); err == nil {
		normalizedEmail := shared.EmailAddress(strings.ToLower(string(emailAddr)))
		acc, err := uc.accountRepo.FindByEmail(ctx, normalizedEmail)
		if err == nil {
			return acc, nil
		}
		if !errors.Is(err, account.ErrAccountNotFound) {
			return nil, err
		}
	}
	return uc.accountRepo.FindByAccountName(ctx, identifier)
}

func (uc *LoginUseCase) checkLoginAttempt(ctx context.Context, ip, identifier string) error {
	if uc.loginLimiter == nil || identifier == "" {
		return nil
	}

	err := uc.loginLimiter.CheckLoginAttempt(ctx, ip, identifier)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domainSecurity.ErrRateLimitExceeded):
		return ErrLoginLocked
	default:
		return ErrLoginFailed
	}
}

func (uc *LoginUseCase) recordFailedLoginAttempt(ctx context.Context, ip, identifier string) {
	if uc.loginLimiter == nil || identifier == "" {
		return
	}
	_ = uc.loginLimiter.RecordLoginAttempt(ctx, ip, identifier, false)
}

func (uc *LoginUseCase) recordSuccessfulLoginAttempt(ctx context.Context, ip, identifier string) {
	if uc.loginLimiter == nil || identifier == "" {
		return
	}
	_ = uc.loginLimiter.RecordLoginAttempt(ctx, ip, identifier, true)
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

func (uc *LoginUseCase) ensurePrimaryUser(ctx context.Context, accountData *account.Account) (shared.UserID, error) {
	if len(accountData.UserIDs) > 0 {
		return accountData.UserIDs[0], nil
	}

	newUser := user.NewUser(accountData.ID, accountData.AccountName)
	userID, err := uc.userRepo.Create(ctx, newUser)
	if err != nil {
		return 0, err
	}
	for _, code := range newUser.RoleCodes {
		if err := uc.userRoleRepo.Assign(ctx, userID, code); err != nil {
			return 0, err
		}
	}
	accountData.AddUser(userID)
	return userID, nil
}

func (uc *LoginUseCase) requiresDeviceVerification(existingDevice *account.AccountDevice, readyOtherDevices int) bool {
	if readyOtherDevices == 0 {
		return false
	}
	if existingDevice == nil {
		return true
	}
	return existingDevice.Status == account.DeviceStatusPendingVerification
}

func (uc *LoginUseCase) directLoginDeviceStatus(existingDevice *account.AccountDevice) account.DeviceStatus {
	if existingDevice == nil {
		return account.DeviceStatusReady
	}
	switch existingDevice.Status {
	case account.DeviceStatusPendingSync, account.DeviceStatusSyncing:
		return existingDevice.Status
	default:
		return account.DeviceStatusReady
	}
}

func (uc *LoginUseCase) ensureNoActiveSelfSync(ctx context.Context, accountData *account.Account, currentDeviceID shared.DeviceID) error {
	if len(accountData.UserIDs) == 0 || uc.selfSyncRepo == nil {
		return nil
	}

	p, err := uc.participantRepo.FindByUserID(ctx, accountData.UserIDs[0])
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return nil
		}
		return ErrLoginFailed
	}
	syncState, err := uc.selfSyncRepo.FindByParticipantID(ctx, p.ID)
	if err != nil {
		if errors.Is(err, selfsenderkeysync.ErrNotFound) {
			return nil
		}
		return ErrLoginFailed
	}
	if syncState.IsActive() && syncState.RequesterDeviceID != currentDeviceID {
		return ErrSelfSenderKeySyncInProgress
	}
	return nil
}

func normalizeLoginIdentifier(identifier string) string {
	trimmed := strings.TrimSpace(identifier)
	if emailAddr, err := shared.ParseEmail(trimmed); err == nil {
		return strings.ToLower(string(emailAddr))
	}
	return trimmed
}

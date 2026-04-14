package usecase

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/auth"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userrole"
)

type VerifyLoginDeviceInput struct {
	Token string
	Code  string
}

type VerifyLoginDeviceOutput struct {
	LoginStatus string
	TokenPair   auth.TokenPair
}

type VerifyLoginDeviceUseCase struct {
	uow               transaction.UnitOfWork
	sessionManager    port.SessionManager
	verificationStore port.VerificationStore
	accountRepo       account.Repository
	userRepo          user.Repository
	userRoleRepo      userrole.Repository
	deviceRepo        device.Repository
	participantRepo   participant.Repository
	selfSyncRepo      selfsenderkeysync.Repository
}

func NewVerifyLoginDeviceUseCase(
	uow transaction.UnitOfWork,
	sessionManager port.SessionManager,
	verificationStore port.VerificationStore,
	accountRepo account.Repository,
	userRepo user.Repository,
	userRoleRepo userrole.Repository,
	deviceRepo device.Repository,
	participantRepo participant.Repository,
	selfSyncRepo selfsenderkeysync.Repository,
) *VerifyLoginDeviceUseCase {
	return &VerifyLoginDeviceUseCase{
		uow:               uow,
		sessionManager:    sessionManager,
		verificationStore: verificationStore,
		accountRepo:       accountRepo,
		userRepo:          userRepo,
		userRoleRepo:      userRoleRepo,
		deviceRepo:        deviceRepo,
		participantRepo:   participantRepo,
		selfSyncRepo:      selfSyncRepo,
	}
}

func (uc *VerifyLoginDeviceUseCase) Execute(
	ctx context.Context,
	input *appShared.UseCaseInput[VerifyLoginDeviceInput],
) (VerifyLoginDeviceOutput, error) {
	session, ok, err := uc.verificationStore.Get(ctx, input.Data.Token)
	if err != nil || !ok {
		return VerifyLoginDeviceOutput{}, ErrTokenInvalid
	}

	if verificationSessionTTL(session) <= 0 || session.ExpiresAtMS <= time.Now().UTC().UnixMilli() {
		_ = uc.verificationStore.Delete(ctx, input.Data.Token)
		return VerifyLoginDeviceOutput{}, ErrTokenInvalid
	}

	if session.Code != strings.TrimSpace(input.Data.Code) {
		remainingAttempts := session.RemainingAttempts - 1
		if remainingAttempts <= 0 {
			if err := uc.verificationStore.Delete(ctx, input.Data.Token); err != nil {
				return VerifyLoginDeviceOutput{}, ErrLoginFailed
			}
			return VerifyLoginDeviceOutput{}, newVerificationAttemptError(ErrVerificationAttemptsExceeded, 0)
		}

		session.RemainingAttempts = remainingAttempts
		if err := uc.verificationStore.Store(ctx, input.Data.Token, session, verificationSessionTTL(session)); err != nil {
			return VerifyLoginDeviceOutput{}, ErrLoginFailed
		}
		return VerifyLoginDeviceOutput{}, newVerificationAttemptError(ErrVerificationCodeInvalid, remainingAttempts)
	}

	if err := uc.verificationStore.Delete(ctx, input.Data.Token); err != nil {
		return VerifyLoginDeviceOutput{}, ErrLoginFailed
	}

	accountData, err := uc.accountRepo.FindByID(ctx, shared.AccountID(session.AccountID))
	if err != nil {
		return VerifyLoginDeviceOutput{}, ErrTokenInvalid
	}
	if err := (&LoginUseCase{}).castStatusError(accountData.Status); err != nil {
		return VerifyLoginDeviceOutput{}, err
	}

	deviceID, err := shared.ParseDeviceID(session.DeviceID)
	if err != nil {
		return VerifyLoginDeviceOutput{}, ErrInvalidDeviceID
	}
	if _, err := uc.deviceRepo.FindByID(ctx, deviceID); err != nil {
		return VerifyLoginDeviceOutput{}, ErrInvalidDeviceID
	}

	ctx, tx, err := uc.uow.Begin(ctx)
	if err != nil {
		return VerifyLoginDeviceOutput{}, ErrLoginFailed
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	primaryUserID, err := uc.ensurePrimaryUser(ctx, accountData)
	if err != nil {
		return VerifyLoginDeviceOutput{}, ErrLoginFailed
	}
	if err := ensureParticipantForUser(ctx, uc.participantRepo, primaryUserID); err != nil {
		return VerifyLoginDeviceOutput{}, ErrLoginFailed
	}
	participantData, err := uc.participantRepo.FindByUserID(ctx, primaryUserID)
	if err != nil {
		return VerifyLoginDeviceOutput{}, ErrLoginFailed
	}
	if err := ensureNoConcurrentSelfSync(ctx, uc.selfSyncRepo, participantData.ID, deviceID); err != nil {
		return VerifyLoginDeviceOutput{}, err
	}

	deviceStatus := account.DeviceStatusReady
	if accountData.CountReadyDevicesExcluding(deviceID) > 0 {
		deviceStatus = account.DeviceStatusPendingSync
	}
	if err := uc.accountRepo.RegisterDevice(ctx, &account.AccountDevice{
		AccountID:  accountData.ID,
		DeviceID:   deviceID,
		Status:     deviceStatus,
		LastIP:     net.ParseIP(session.IP),
		LastSeenAt: time.Now(),
	}); err != nil {
		return VerifyLoginDeviceOutput{}, ErrLoginFailed
	}

	if deviceStatus == account.DeviceStatusPendingSync {
		if _, err := uc.selfSyncRepo.UpsertPending(ctx, participantData.ID, deviceID); err != nil {
			return VerifyLoginDeviceOutput{}, ErrLoginFailed
		}
	}

	tokenPair, err := uc.sessionManager.Create(ctx, auth.CreateSessionInput{
		AccountID: accountData.ID,
		UserID:    primaryUserID,
		DeviceID:  deviceID,
	})
	if err != nil {
		return VerifyLoginDeviceOutput{}, ErrLoginFailed
	}

	if err := uc.accountRepo.RecordLoginEvent(ctx, &account.AccountLoginEvent{
		AccountID: accountData.ID,
		DeviceID:  deviceID,
		IPAddress: net.ParseIP(session.IP),
		UserAgent: session.UserAgent,
		Success:   true,
	}); err != nil {
		return VerifyLoginDeviceOutput{}, ErrLoginFailed
	}
	if err := uc.accountRepo.Update(ctx, accountData); err != nil {
		return VerifyLoginDeviceOutput{}, ErrLoginFailed
	}
	if err := tx.Commit(); err != nil {
		return VerifyLoginDeviceOutput{}, ErrLoginFailed
	}
	committed = true

	return VerifyLoginDeviceOutput{
		LoginStatus: "authenticated",
		TokenPair:   tokenPair,
	}, nil
}

func (uc *VerifyLoginDeviceUseCase) ensurePrimaryUser(ctx context.Context, accountData *account.Account) (shared.UserID, error) {
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

func ensureParticipantForUser(ctx context.Context, repo participant.Repository, userID shared.UserID) error {
	_, err := repo.FindByUserID(ctx, userID)
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
	if err := repo.Create(ctx, &p); err != nil && !errors.Is(err, participant.ErrAlreadyExists) {
		return err
	}
	return nil
}

func ensureNoConcurrentSelfSync(
	ctx context.Context,
	repo selfsenderkeysync.Repository,
	participantID participant.ID,
	currentDeviceID shared.DeviceID,
) error {
	if repo == nil {
		return nil
	}
	syncState, err := repo.FindByParticipantID(ctx, participantID)
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

package usecase

import (
	"context"
	"errors"
	"time"

	e2eePort "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysyncdistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"go.uber.org/zap"
)

type AcceptSelfSenderKeySyncInput struct{}
type AcceptSelfSenderKeySyncOutput = SelfSenderKeySyncSnapshot

type MarkSelfSenderKeySyncUploadedInput struct{}
type MarkSelfSenderKeySyncUploadedOutput = SelfSenderKeySyncSnapshot

type CompleteSelfSenderKeySyncInput struct{}
type CompleteSelfSenderKeySyncOutput = SelfSenderKeySyncSnapshot

type FailSelfSenderKeySyncInput struct {
	LastError string
	Retryable bool
}
type FailSelfSenderKeySyncOutput = SelfSenderKeySyncSnapshot

type SelfSenderKeySyncMutationUseCase struct {
	participantRepo participant.Repository
	selfSyncRepo    selfsenderkeysync.Repository
	selfSyncCopyRepo selfsenderkeysyncdistribution.Repository
	accountRepo     account.Repository
	deviceRepo      device.Repository
	broadcaster     e2eePort.Broadcaster
}

func NewSelfSenderKeySyncMutationUseCase(
	participantRepo participant.Repository,
	selfSyncRepo selfsenderkeysync.Repository,
	selfSyncCopyRepo selfsenderkeysyncdistribution.Repository,
	accountRepo account.Repository,
	deviceRepo device.Repository,
	broadcaster e2eePort.Broadcaster,
) *SelfSenderKeySyncMutationUseCase {
	return &SelfSenderKeySyncMutationUseCase{
		participantRepo: participantRepo,
		selfSyncRepo:    selfSyncRepo,
		selfSyncCopyRepo: selfSyncCopyRepo,
		accountRepo:     accountRepo,
		deviceRepo:      deviceRepo,
		broadcaster:     broadcaster,
	}
}

func (u *SelfSenderKeySyncMutationUseCase) Accept(
	ctx context.Context,
	input appShared.UseCaseInput[AcceptSelfSenderKeySyncInput],
) (*AcceptSelfSenderKeySyncOutput, error) {
	participantData, syncState, err := u.loadSync(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, err
	}
	if syncState.RequesterDeviceID == input.Base.Request.DeviceID {
		return nil, ErrForbidden
	}

	claimed, err := u.selfSyncRepo.ClaimProvider(ctx, syncState.ID, participantData.ID, syncState.RequesterDeviceID, input.Base.Request.DeviceID)
	if err != nil || !claimed {
		return nil, ErrForbidden
	}
	fallbackState := acceptedSelfSenderKeySync(syncState, input.Base.Request.DeviceID)
	if err := u.accountRepo.UpdateDeviceStatus(ctx, input.Base.Auth.AccountID, syncState.RequesterDeviceID, account.DeviceStatusSyncing); err != nil {
		logger.Log.Warn("self sender key sync accept updated state but failed to mark requester device syncing",
			zap.Int64("participant_id", int64(participantData.ID)),
			zap.String("requester_device_id", syncState.RequesterDeviceID.String()),
			zap.Error(err),
		)
	}
	return u.snapshotAndBroadcastAfterMutation(ctx, input.Base.Auth.AccountID, input.Base.Auth.UserID, input.Base.Request.DeviceID, participantData.ID, fallbackState)
}

func (u *SelfSenderKeySyncMutationUseCase) MarkUploaded(
	ctx context.Context,
	input appShared.UseCaseInput[MarkSelfSenderKeySyncUploadedInput],
) (*MarkSelfSenderKeySyncUploadedOutput, error) {
	participantData, syncState, err := u.loadSync(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, err
	}
	if syncState.ProviderDeviceID == nil || *syncState.ProviderDeviceID != input.Base.Request.DeviceID {
		return nil, ErrForbidden
	}
	if err := u.selfSyncRepo.MarkUploaded(ctx, syncState.ID, participantData.ID, syncState.RequesterDeviceID, input.Base.Request.DeviceID); err != nil {
		return nil, ErrForbidden
	}
	return u.snapshotAndBroadcastAfterMutation(
		ctx,
		input.Base.Auth.AccountID,
		input.Base.Auth.UserID,
		input.Base.Request.DeviceID,
		participantData.ID,
		uploadedSelfSenderKeySync(syncState),
	)
}

func (u *SelfSenderKeySyncMutationUseCase) Complete(
	ctx context.Context,
	input appShared.UseCaseInput[CompleteSelfSenderKeySyncInput],
) (*CompleteSelfSenderKeySyncOutput, error) {
	participantData, syncState, err := u.loadSync(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, err
	}
	if syncState.RequesterDeviceID != input.Base.Request.DeviceID {
		return nil, ErrForbidden
	}
	hasNonConsumed, err := u.selfSyncCopyRepo.HasNonConsumed(ctx, syncState.ID, participantData.ID, input.Base.Request.DeviceID)
	if err != nil {
		return nil, err
	}
	if hasNonConsumed {
		return nil, ErrSelfSenderKeySyncIncomplete
	}
	if err := u.selfSyncRepo.MarkCompleted(ctx, syncState.ID, participantData.ID, input.Base.Request.DeviceID); err != nil {
		return nil, ErrForbidden
	}
	fallbackState := completedSelfSenderKeySync(syncState)
	if err := u.accountRepo.UpdateDeviceStatus(ctx, input.Base.Auth.AccountID, input.Base.Request.DeviceID, account.DeviceStatusReady); err != nil {
		logger.Log.Warn("self sender key sync complete updated state but failed to mark requester device ready",
			zap.Int64("participant_id", int64(participantData.ID)),
			zap.String("requester_device_id", input.Base.Request.DeviceID.String()),
			zap.Error(err),
		)
	}
	return u.snapshotAndBroadcastAfterMutation(
		ctx,
		input.Base.Auth.AccountID,
		input.Base.Auth.UserID,
		input.Base.Request.DeviceID,
		participantData.ID,
		fallbackState,
	)
}

func (u *SelfSenderKeySyncMutationUseCase) Fail(
	ctx context.Context,
	input appShared.UseCaseInput[FailSelfSenderKeySyncInput],
) (*FailSelfSenderKeySyncOutput, error) {
	participantData, syncState, err := u.loadSync(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, err
	}

	providerDeviceID := shared.DeviceID{}
	if syncState.ProviderDeviceID != nil {
		providerDeviceID = *syncState.ProviderDeviceID
	}
	if syncState.RequesterDeviceID != input.Base.Request.DeviceID && providerDeviceID != input.Base.Request.DeviceID {
		return nil, ErrForbidden
	}
	if err := u.selfSyncRepo.MarkFailed(ctx, syncState.ID, participantData.ID, syncState.RequesterDeviceID, providerDeviceID, input.Data.LastError, input.Data.Retryable); err != nil {
		return nil, ErrForbidden
	}
	fallbackState := failedSelfSenderKeySync(syncState, input.Data.LastError, input.Data.Retryable)
	if input.Data.Retryable {
		if err := u.accountRepo.UpdateDeviceStatus(ctx, input.Base.Auth.AccountID, syncState.RequesterDeviceID, account.DeviceStatusPendingSync); err != nil {
			logger.Log.Warn("self sender key sync fail updated state but failed to mark requester device pending sync",
				zap.Int64("participant_id", int64(participantData.ID)),
				zap.String("requester_device_id", syncState.RequesterDeviceID.String()),
				zap.Error(err),
			)
		}
	}
	return u.snapshotAndBroadcastAfterMutation(
		ctx,
		input.Base.Auth.AccountID,
		input.Base.Auth.UserID,
		input.Base.Request.DeviceID,
		participantData.ID,
		fallbackState,
	)
}

func (u *SelfSenderKeySyncMutationUseCase) loadSync(
	ctx context.Context,
	userID shared.UserID,
) (*participant.Participant, *selfsenderkeysync.SelfSenderKeySync, error) {
	participantData, err := loadCurrentParticipant(ctx, u.participantRepo, userID)
	if err != nil {
		return nil, nil, ErrForbidden
	}
	syncState, err := u.selfSyncRepo.FindByParticipantID(ctx, participantData.ID)
	if err != nil {
		if errors.Is(err, selfsenderkeysync.ErrNotFound) {
			return nil, nil, ErrForbidden
		}
		return nil, nil, ErrForbidden
	}
	return participantData, syncState, nil
}

func (u *SelfSenderKeySyncMutationUseCase) snapshotAndBroadcast(
	ctx context.Context,
	accountID shared.AccountID,
	userID shared.UserID,
	currentDeviceID shared.DeviceID,
	participantID participant.ID,
) (*SelfSenderKeySyncSnapshot, error) {
	syncState, err := u.selfSyncRepo.FindByParticipantID(ctx, participantID)
	if err != nil {
		return nil, ErrForbidden
	}
	snapshot, err := buildSelfSenderKeySyncSnapshot(ctx, u.accountRepo, u.deviceRepo, accountID, currentDeviceID, syncState)
	if err != nil {
		return nil, ErrForbidden
	}
	broadcastSelfSenderKeySyncStateChanged(u.broadcaster, userID, snapshot)
	return snapshot, nil
}

func (u *SelfSenderKeySyncMutationUseCase) snapshotAndBroadcastAfterMutation(
	ctx context.Context,
	accountID shared.AccountID,
	userID shared.UserID,
	currentDeviceID shared.DeviceID,
	participantID participant.ID,
	fallbackState *selfsenderkeysync.SelfSenderKeySync,
) (*SelfSenderKeySyncSnapshot, error) {
	snapshot, err := u.snapshotAndBroadcast(ctx, accountID, userID, currentDeviceID, participantID)
	if err == nil {
		return snapshot, nil
	}
	if fallbackState == nil {
		return nil, err
	}

	logger.Log.Warn("self sender key sync mutation succeeded but snapshot refresh failed; returning fallback snapshot",
		zap.Int64("participant_id", int64(participantID)),
		zap.Error(err),
	)
	fallbackSnapshot := fallbackSelfSenderKeySyncSnapshot(currentDeviceID, fallbackState)
	broadcastSelfSenderKeySyncStateChanged(u.broadcaster, userID, fallbackSnapshot)
	return fallbackSnapshot, nil
}

func fallbackSelfSenderKeySyncSnapshot(
	currentDeviceID shared.DeviceID,
	syncState *selfsenderkeysync.SelfSenderKeySync,
) *SelfSenderKeySyncSnapshot {
	snapshot := &SelfSenderKeySyncSnapshot{
		Exists:                 syncState != nil,
		Status:                 "idle",
		RequesterCurrentDevice: false,
		ProviderCurrentDevice:  false,
	}
	if syncState == nil {
		return snapshot
	}

	snapshot.Status = string(syncState.Status)
	snapshot.RequesterCurrentDevice = syncState.RequesterDeviceID == currentDeviceID
	snapshot.RequesterDevice = &SelfSenderKeySyncDeviceSnapshot{
		DeviceID: syncState.RequesterDeviceID.String(),
	}
	snapshot.RequestedAtMS = syncState.RequestedAt.UnixMilli()
	if syncState.ProviderDeviceID != nil {
		snapshot.ProviderCurrentDevice = *syncState.ProviderDeviceID == currentDeviceID
		snapshot.ProviderDevice = &SelfSenderKeySyncDeviceSnapshot{
			DeviceID: syncState.ProviderDeviceID.String(),
		}
	}
	if syncState.ProviderClaimedAt != nil {
		snapshot.ProviderClaimedAtMS = syncState.ProviderClaimedAt.UnixMilli()
	}
	if syncState.UploadedAt != nil {
		snapshot.UploadedAtMS = syncState.UploadedAt.UnixMilli()
	}
	if syncState.CompletedAt != nil {
		snapshot.CompletedAtMS = syncState.CompletedAt.UnixMilli()
	}
	if syncState.FailedAt != nil {
		snapshot.FailedAtMS = syncState.FailedAt.UnixMilli()
	}
	if syncState.LastError != nil {
		snapshot.LastError = *syncState.LastError
	}
	return snapshot
}

func acceptedSelfSenderKeySync(
	syncState *selfsenderkeysync.SelfSenderKeySync,
	providerDeviceID shared.DeviceID,
) *selfsenderkeysync.SelfSenderKeySync {
	next := cloneSelfSenderKeySync(syncState)
	now := time.Now()
	next.ProviderDeviceID = &providerDeviceID
	next.Status = selfsenderkeysync.StatusSyncing
	next.ProviderClaimedAt = &now
	next.FailedAt = nil
	next.LastError = nil
	next.UpdatedAt = now
	return next
}

func uploadedSelfSenderKeySync(syncState *selfsenderkeysync.SelfSenderKeySync) *selfsenderkeysync.SelfSenderKeySync {
	next := cloneSelfSenderKeySync(syncState)
	now := time.Now()
	next.Status = selfsenderkeysync.StatusUploaded
	next.UploadedAt = &now
	next.FailedAt = nil
	next.LastError = nil
	next.UpdatedAt = now
	return next
}

func completedSelfSenderKeySync(syncState *selfsenderkeysync.SelfSenderKeySync) *selfsenderkeysync.SelfSenderKeySync {
	next := cloneSelfSenderKeySync(syncState)
	now := time.Now()
	next.Status = selfsenderkeysync.StatusCompleted
	next.CompletedAt = &now
	next.FailedAt = nil
	next.LastError = nil
	next.UpdatedAt = now
	return next
}

func failedSelfSenderKeySync(
	syncState *selfsenderkeysync.SelfSenderKeySync,
	lastError string,
	retryable bool,
) *selfsenderkeysync.SelfSenderKeySync {
	next := cloneSelfSenderKeySync(syncState)
	now := time.Now()
	if retryable {
		next.Status = selfsenderkeysync.StatusPendingProvider
		next.ProviderDeviceID = nil
	} else {
		next.Status = selfsenderkeysync.StatusFailed
	}
	next.FailedAt = &now
	if lastError != "" {
		errCopy := lastError
		next.LastError = &errCopy
	} else {
		next.LastError = nil
	}
	next.UpdatedAt = now
	return next
}

func cloneSelfSenderKeySync(syncState *selfsenderkeysync.SelfSenderKeySync) *selfsenderkeysync.SelfSenderKeySync {
	if syncState == nil {
		return nil
	}
	cloned := *syncState
	if syncState.ProviderDeviceID != nil {
		deviceID := *syncState.ProviderDeviceID
		cloned.ProviderDeviceID = &deviceID
	}
	if syncState.ProviderClaimedAt != nil {
		timestamp := *syncState.ProviderClaimedAt
		cloned.ProviderClaimedAt = &timestamp
	}
	if syncState.UploadedAt != nil {
		timestamp := *syncState.UploadedAt
		cloned.UploadedAt = &timestamp
	}
	if syncState.CompletedAt != nil {
		timestamp := *syncState.CompletedAt
		cloned.CompletedAt = &timestamp
	}
	if syncState.FailedAt != nil {
		timestamp := *syncState.FailedAt
		cloned.FailedAt = &timestamp
	}
	if syncState.LastError != nil {
		errText := *syncState.LastError
		cloned.LastError = &errText
	}
	return &cloned
}

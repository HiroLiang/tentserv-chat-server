package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysyncdistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSelfSenderKeySyncMutationUseCaseAcceptReturnsFallbackSnapshotAfterMutationSuccess(t *testing.T) {
	logger.Log = zap.NewNop()
	requesterDeviceID := mustParseSyncTestDeviceID(t, "11111111-1111-4111-8111-111111111111")
	providerDeviceID := mustParseSyncTestDeviceID(t, "22222222-2222-4222-8222-222222222222")
	uc, selfSyncRepo, broadcaster := newSelfSyncMutationUseCaseForFallbackTests(&selfsenderkeysync.SelfSenderKeySync{
		ParticipantID:     3001,
		RequesterDeviceID: requesterDeviceID,
		Status:            selfsenderkeysync.StatusPendingProvider,
		RequestedAt:       time.Now().Add(-time.Minute),
		UpdatedAt:         time.Now().Add(-time.Minute),
	})

	out, err := uc.Accept(context.Background(), appShared.UseCaseInput[AcceptSelfSenderKeySyncInput]{
		Base: syncMutationBaseContext(901, 801, providerDeviceID),
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, out.Exists)
	assert.Equal(t, string(selfsenderkeysync.StatusSyncing), out.Status)
	require.NotNil(t, out.RequesterDevice)
	assert.Equal(t, requesterDeviceID.String(), out.RequesterDevice.DeviceID)
	require.NotNil(t, out.ProviderDevice)
	assert.Equal(t, providerDeviceID.String(), out.ProviderDevice.DeviceID)
	assert.False(t, out.RequesterCurrentDevice)
	assert.True(t, out.ProviderCurrentDevice)
	assert.NotNil(t, selfSyncRepo.current.ProviderDeviceID)
	assert.Equal(t, providerDeviceID, *selfSyncRepo.current.ProviderDeviceID)
	assert.Equal(t, selfsenderkeysync.StatusSyncing, selfSyncRepo.current.Status)
	require.Len(t, broadcaster.payloads, 1)
	assertSelfSyncBroadcastStatus(t, broadcaster.payloads[0], string(selfsenderkeysync.StatusSyncing))
}

func TestSelfSenderKeySyncMutationUseCaseCompleteReturnsFallbackSnapshotAfterMutationSuccess(t *testing.T) {
	logger.Log = zap.NewNop()
	requesterDeviceID := mustParseSyncTestDeviceID(t, "33333333-3333-4333-8333-333333333333")
	providerDeviceID := mustParseSyncTestDeviceID(t, "44444444-4444-4444-8444-444444444444")
	uc, selfSyncRepo, broadcaster := newSelfSyncMutationUseCaseForFallbackTests(&selfsenderkeysync.SelfSenderKeySync{
		ParticipantID:     3002,
		RequesterDeviceID: requesterDeviceID,
		ProviderDeviceID:  &providerDeviceID,
		Status:            selfsenderkeysync.StatusUploaded,
		RequestedAt:       time.Now().Add(-2 * time.Minute),
		UpdatedAt:         time.Now().Add(-time.Minute),
	})

	out, err := uc.Complete(context.Background(), appShared.UseCaseInput[CompleteSelfSenderKeySyncInput]{
		Base: syncMutationBaseContext(902, 802, requesterDeviceID),
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, string(selfsenderkeysync.StatusCompleted), out.Status)
	assert.True(t, out.RequesterCurrentDevice)
	assert.False(t, out.ProviderCurrentDevice)
	assert.Equal(t, selfsenderkeysync.StatusCompleted, selfSyncRepo.current.Status)
	require.Len(t, broadcaster.payloads, 1)
	assertSelfSyncBroadcastStatus(t, broadcaster.payloads[0], string(selfsenderkeysync.StatusCompleted))
}

func TestSelfSenderKeySyncMutationUseCaseCompleteRejectsWhenAnyCopyIsNotConsumed(t *testing.T) {
	logger.Log = zap.NewNop()
	requesterDeviceID := mustParseSyncTestDeviceID(t, "99999999-9999-4999-8999-999999999999")
	providerDeviceID := mustParseSyncTestDeviceID(t, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	selfSyncRepo := &selfSyncRepoFallbackStub{current: cloneSelfSenderKeySync(&selfsenderkeysync.SelfSenderKeySync{
		ID:                44,
		ParticipantID:     3004,
		RequesterDeviceID: requesterDeviceID,
		ProviderDeviceID:  &providerDeviceID,
		Status:            selfsenderkeysync.StatusUploaded,
		RequestedAt:       time.Now().Add(-2 * time.Minute),
		UpdatedAt:         time.Now().Add(-time.Minute),
	})}
	copyRepo := &selfSyncCopyRepoFallbackStub{hasNonConsumed: true}
	uc := NewSelfSenderKeySyncMutationUseCase(
		&selfSyncParticipantRepoStub{
			participant: &participant.Participant{
				ID:     3004,
				Type:   participant.UserType,
				UserID: ptrSharedUserID(804),
			},
		},
		selfSyncRepo,
		copyRepo,
		&selfSyncAccountRepoFallbackStub{},
		&selfSyncDeviceRepoFallbackStub{},
		&selfSyncBroadcasterStub{},
	)

	out, err := uc.Complete(context.Background(), appShared.UseCaseInput[CompleteSelfSenderKeySyncInput]{
		Base: syncMutationBaseContext(904, 804, requesterDeviceID),
	})

	require.ErrorIs(t, err, ErrSelfSenderKeySyncIncomplete)
	assert.Nil(t, out)
	assert.Equal(t, selfsenderkeysync.StatusUploaded, selfSyncRepo.current.Status)
}

func TestSelfSenderKeySyncMutationUseCaseFailReturnsFallbackSnapshotAfterMutationSuccess(t *testing.T) {
	logger.Log = zap.NewNop()
	requesterDeviceID := mustParseSyncTestDeviceID(t, "55555555-5555-4555-8555-555555555555")
	providerDeviceID := mustParseSyncTestDeviceID(t, "66666666-6666-4666-8666-666666666666")
	uc, selfSyncRepo, broadcaster := newSelfSyncMutationUseCaseForFallbackTests(&selfsenderkeysync.SelfSenderKeySync{
		ParticipantID:     3003,
		RequesterDeviceID: requesterDeviceID,
		ProviderDeviceID:  &providerDeviceID,
		Status:            selfsenderkeysync.StatusSyncing,
		RequestedAt:       time.Now().Add(-2 * time.Minute),
		ProviderClaimedAt: timePtr(time.Now().Add(-time.Minute)),
		UpdatedAt:         time.Now().Add(-time.Minute),
	})

	out, err := uc.Fail(context.Background(), appShared.UseCaseInput[FailSelfSenderKeySyncInput]{
		Base: syncMutationBaseContext(903, 803, requesterDeviceID),
		Data: FailSelfSenderKeySyncInput{
			LastError: "provider upload timed out",
			Retryable: true,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, string(selfsenderkeysync.StatusPendingProvider), out.Status)
	assert.Equal(t, "provider upload timed out", out.LastError)
	assert.Nil(t, selfSyncRepo.current.ProviderDeviceID)
	assert.Equal(t, selfsenderkeysync.StatusPendingProvider, selfSyncRepo.current.Status)
	require.Len(t, broadcaster.payloads, 1)
	assertSelfSyncBroadcastStatus(t, broadcaster.payloads[0], string(selfsenderkeysync.StatusPendingProvider))
}

func newSelfSyncMutationUseCaseForFallbackTests(
	initial *selfsenderkeysync.SelfSenderKeySync,
) (*SelfSenderKeySyncMutationUseCase, *selfSyncRepoFallbackStub, *selfSyncBroadcasterStub) {
	repo := &selfSyncRepoFallbackStub{current: cloneSelfSenderKeySync(initial)}
	broadcaster := &selfSyncBroadcasterStub{}
	uc := NewSelfSenderKeySyncMutationUseCase(
		&selfSyncParticipantRepoStub{
			participant: &participant.Participant{
				ID:     initial.ParticipantID,
				Type:   participant.UserType,
				UserID: ptrSharedUserID(801),
			},
		},
		repo,
		&selfSyncCopyRepoFallbackStub{},
		&selfSyncAccountRepoFallbackStub{},
		&selfSyncDeviceRepoFallbackStub{},
		broadcaster,
	)
	return uc, repo, broadcaster
}

func syncMutationBaseContext(accountID shared.AccountID, userID shared.UserID, deviceID shared.DeviceID) appShared.BaseContext {
	return appShared.BaseContext{
		Request: appShared.RequestContext{DeviceID: deviceID},
		Auth: &appShared.AuthContext{
			AccountID: accountID,
			UserID:    userID,
		},
	}
}

func assertSelfSyncBroadcastStatus(t *testing.T, payload []byte, expected string) {
	t.Helper()
	var envelope struct {
		Type    string `json:"type"`
		Payload struct {
			Status string `json:"status"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(payload, &envelope))
	assert.Equal(t, "e2ee.self_sender_key_sync_state_changed", envelope.Type)
	assert.Equal(t, expected, envelope.Payload.Status)
}

func mustParseSyncTestDeviceID(t *testing.T, value string) shared.DeviceID {
	t.Helper()
	deviceID, err := shared.ParseDeviceID(value)
	require.NoError(t, err)
	return deviceID
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func ptrSharedUserID(value shared.UserID) *shared.UserID {
	return &value
}

type selfSyncParticipantRepoStub struct {
	participant *participant.Participant
}

func (s *selfSyncParticipantRepoStub) FindByID(context.Context, participant.ID) (*participant.Participant, error) {
	if s.participant == nil {
		return nil, participant.ErrNotFound
	}
	copied := *s.participant
	return &copied, nil
}

func (s *selfSyncParticipantRepoStub) FindByUserID(context.Context, shared.UserID) (*participant.Participant, error) {
	if s.participant == nil {
		return nil, participant.ErrNotFound
	}
	copied := *s.participant
	return &copied, nil
}

func (s *selfSyncParticipantRepoStub) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *selfSyncParticipantRepoStub) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *selfSyncParticipantRepoStub) Create(context.Context, *participant.Participant) error {
	return nil
}

type selfSyncRepoFallbackStub struct {
	current *selfsenderkeysync.SelfSenderKeySync
}

func (s *selfSyncRepoFallbackStub) FindByParticipantID(context.Context, participant.ID) (*selfsenderkeysync.SelfSenderKeySync, error) {
	if s.current == nil {
		return nil, selfsenderkeysync.ErrNotFound
	}
	return cloneSelfSenderKeySync(s.current), nil
}

func (s *selfSyncRepoFallbackStub) UpsertPending(context.Context, participant.ID, shared.DeviceID) (*selfsenderkeysync.SelfSenderKeySync, error) {
	return nil, nil
}

func (s *selfSyncRepoFallbackStub) ClaimProvider(_ context.Context, id selfsenderkeysync.ID, participantID participant.ID, requesterDeviceID, providerDeviceID shared.DeviceID) (bool, error) {
	if s.current == nil || s.current.ID != id || s.current.ParticipantID != participantID || s.current.RequesterDeviceID != requesterDeviceID || s.current.Status != selfsenderkeysync.StatusPendingProvider {
		return false, nil
	}
	now := time.Now()
	s.current.ProviderDeviceID = &providerDeviceID
	s.current.Status = selfsenderkeysync.StatusSyncing
	s.current.ProviderClaimedAt = &now
	s.current.UpdatedAt = now
	return true, nil
}

func (s *selfSyncRepoFallbackStub) MarkUploaded(_ context.Context, id selfsenderkeysync.ID, participantID participant.ID, requesterDeviceID, providerDeviceID shared.DeviceID) error {
	if s.current == nil || s.current.ID != id || s.current.ParticipantID != participantID || s.current.RequesterDeviceID != requesterDeviceID || s.current.ProviderDeviceID == nil || *s.current.ProviderDeviceID != providerDeviceID {
		return selfsenderkeysync.ErrNotFound
	}
	now := time.Now()
	s.current.Status = selfsenderkeysync.StatusUploaded
	s.current.UploadedAt = &now
	s.current.UpdatedAt = now
	return nil
}

func (s *selfSyncRepoFallbackStub) MarkCompleted(_ context.Context, id selfsenderkeysync.ID, participantID participant.ID, requesterDeviceID shared.DeviceID) error {
	if s.current == nil || s.current.ID != id || s.current.ParticipantID != participantID || s.current.RequesterDeviceID != requesterDeviceID {
		return selfsenderkeysync.ErrNotFound
	}
	now := time.Now()
	s.current.Status = selfsenderkeysync.StatusCompleted
	s.current.CompletedAt = &now
	s.current.UpdatedAt = now
	return nil
}

func (s *selfSyncRepoFallbackStub) MarkFailed(_ context.Context, id selfsenderkeysync.ID, participantID participant.ID, requesterDeviceID, providerDeviceID shared.DeviceID, lastError string, retryable bool) error {
	if s.current == nil || s.current.ID != id || s.current.ParticipantID != participantID || s.current.RequesterDeviceID != requesterDeviceID {
		return selfsenderkeysync.ErrNotFound
	}
	now := time.Now()
	if retryable {
		s.current.Status = selfsenderkeysync.StatusPendingProvider
		s.current.ProviderDeviceID = nil
	} else {
		s.current.Status = selfsenderkeysync.StatusFailed
	}
	s.current.FailedAt = &now
	s.current.UpdatedAt = now
	if lastError != "" {
		errorCopy := lastError
		s.current.LastError = &errorCopy
	}
	return nil
}

type selfSyncAccountRepoFallbackStub struct{}

type selfSyncCopyRepoFallbackStub struct {
	hasNonConsumed bool
}

func (s *selfSyncCopyRepoFallbackStub) ReplaceForSync(context.Context, selfsenderkeysync.ID, participant.ID, shared.DeviceID, shared.DeviceID, []*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution) error {
	return nil
}

func (s *selfSyncCopyRepoFallbackStub) FindPendingByRequester(context.Context, selfsenderkeysync.ID, participant.ID, shared.DeviceID) ([]*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution, error) {
	return nil, nil
}

func (s *selfSyncCopyRepoFallbackStub) MarkConsumed(context.Context, selfsenderkeysync.ID, selfsenderkeysyncdistribution.ID, participant.ID, shared.DeviceID) error {
	return nil
}

func (s *selfSyncCopyRepoFallbackStub) MarkFailed(context.Context, selfsenderkeysync.ID, selfsenderkeysyncdistribution.ID, participant.ID, shared.DeviceID) error {
	return nil
}

func (s *selfSyncCopyRepoFallbackStub) HasNonConsumed(context.Context, selfsenderkeysync.ID, participant.ID, shared.DeviceID) (bool, error) {
	return s.hasNonConsumed, nil
}

func (s *selfSyncAccountRepoFallbackStub) FindByID(context.Context, shared.AccountID) (*account.Account, error) {
	return nil, account.ErrAccountNotFound
}

func (s *selfSyncAccountRepoFallbackStub) FindByAccountName(context.Context, string) (*account.Account, error) {
	return nil, account.ErrAccountNotFound
}

func (s *selfSyncAccountRepoFallbackStub) FindByEmail(context.Context, shared.EmailAddress) (*account.Account, error) {
	return nil, account.ErrAccountNotFound
}

func (s *selfSyncAccountRepoFallbackStub) Create(context.Context, *account.Account) (shared.AccountID, error) {
	return 0, nil
}

func (s *selfSyncAccountRepoFallbackStub) Update(context.Context, *account.Account) error {
	return nil
}

func (s *selfSyncAccountRepoFallbackStub) RegisterDevice(context.Context, *account.AccountDevice) error {
	return nil
}

func (s *selfSyncAccountRepoFallbackStub) UpdateDeviceStatus(context.Context, shared.AccountID, shared.DeviceID, account.DeviceStatus) error {
	return nil
}

func (s *selfSyncAccountRepoFallbackStub) RecordLoginEvent(context.Context, *account.AccountLoginEvent) error {
	return nil
}

func (s *selfSyncAccountRepoFallbackStub) ReplaceDevices(context.Context, shared.AccountID, []account.AccountDevice) error {
	return nil
}

type selfSyncDeviceRepoFallbackStub struct{}

func (s *selfSyncDeviceRepoFallbackStub) FindByID(context.Context, shared.DeviceID) (*device.Device, error) {
	return nil, device.ErrDeviceNotFound
}

func (s *selfSyncDeviceRepoFallbackStub) FindAllByAccountID(context.Context, shared.AccountID) ([]*device.Device, error) {
	return nil, nil
}

func (s *selfSyncDeviceRepoFallbackStub) Create(context.Context, *device.Device) error {
	return nil
}

func (s *selfSyncDeviceRepoFallbackStub) Update(context.Context, *device.Device) error {
	return nil
}

func (s *selfSyncDeviceRepoFallbackStub) BindAccount(context.Context, shared.DeviceID, shared.AccountID) error {
	return nil
}

func (s *selfSyncDeviceRepoFallbackStub) DeleteByAccount(context.Context, shared.DeviceID, shared.AccountID) error {
	return nil
}

type selfSyncBroadcasterStub struct {
	payloads [][]byte
}

func (s *selfSyncBroadcasterStub) SendToUser(_ string, payload []byte) {
	copied := make([]byte, len(payload))
	copy(copied, payload)
	s.payloads = append(s.payloads, copied)
}

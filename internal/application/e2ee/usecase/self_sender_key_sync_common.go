package usecase

import (
	"context"
	"encoding/json"
	"strconv"

	e2eePort "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type SelfSenderKeySyncDeviceSnapshot struct {
	DeviceID      string `json:"device_id"`
	DeviceName    string `json:"device_name"`
	Platform      string `json:"platform"`
	LastIP        string `json:"last_ip,omitempty"`
	BindingStatus string `json:"binding_status,omitempty"`
}

type SelfSenderKeySyncSnapshot struct {
	Exists                 bool                             `json:"exists"`
	Status                 string                           `json:"status"`
	RequesterDevice        *SelfSenderKeySyncDeviceSnapshot `json:"requester_device,omitempty"`
	ProviderDevice         *SelfSenderKeySyncDeviceSnapshot `json:"provider_device,omitempty"`
	RequesterCurrentDevice bool                             `json:"requester_current_device"`
	ProviderCurrentDevice  bool                             `json:"provider_current_device"`
	LastError              string                           `json:"last_error,omitempty"`
	RequestedAtMS          int64                            `json:"requested_at_ms,omitempty"`
	ProviderClaimedAtMS    int64                            `json:"provider_claimed_at_ms,omitempty"`
	UploadedAtMS           int64                            `json:"uploaded_at_ms,omitempty"`
	CompletedAtMS          int64                            `json:"completed_at_ms,omitempty"`
	FailedAtMS             int64                            `json:"failed_at_ms,omitempty"`
}

func buildSelfSenderKeySyncSnapshot(
	ctx context.Context,
	accountRepo account.Repository,
	deviceRepo device.Repository,
	accountID shared.AccountID,
	currentDeviceID shared.DeviceID,
	syncState *selfsenderkeysync.SelfSenderKeySync,
) (*SelfSenderKeySyncSnapshot, error) {
	snapshot := &SelfSenderKeySyncSnapshot{
		Exists: false,
		Status: "idle",
	}
	if syncState == nil {
		return snapshot, nil
	}

	accountData, err := accountRepo.FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	snapshot.Exists = true
	snapshot.Status = string(syncState.Status)
	snapshot.RequesterCurrentDevice = syncState.RequesterDeviceID == currentDeviceID
	snapshot.RequestedAtMS = syncState.RequestedAt.UnixMilli()
	if syncState.ProviderDeviceID != nil {
		snapshot.ProviderCurrentDevice = *syncState.ProviderDeviceID == currentDeviceID
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

	requesterDevice, _ := deviceRepo.FindByID(ctx, syncState.RequesterDeviceID)
	snapshot.RequesterDevice = toSelfSenderKeySyncDeviceSnapshot(requesterDevice, accountData.GetDevice(syncState.RequesterDeviceID))
	if syncState.ProviderDeviceID != nil {
		providerDevice, _ := deviceRepo.FindByID(ctx, *syncState.ProviderDeviceID)
		snapshot.ProviderDevice = toSelfSenderKeySyncDeviceSnapshot(providerDevice, accountData.GetDevice(*syncState.ProviderDeviceID))
	}

	return snapshot, nil
}

func toSelfSenderKeySyncDeviceSnapshot(
	deviceData *device.Device,
	binding *account.AccountDevice,
) *SelfSenderKeySyncDeviceSnapshot {
	if deviceData == nil && binding == nil {
		return nil
	}

	snapshot := &SelfSenderKeySyncDeviceSnapshot{}
	if deviceData != nil {
		snapshot.DeviceID = deviceData.ID.String()
		snapshot.DeviceName = deviceData.Name
		snapshot.Platform = string(deviceData.Platform)
	}
	if binding != nil {
		snapshot.BindingStatus = string(binding.Status)
		if binding.LastIP != nil {
			snapshot.LastIP = binding.LastIP.String()
		}
		if snapshot.DeviceID == "" {
			snapshot.DeviceID = binding.DeviceID.String()
		}
	}
	return snapshot
}

func broadcastSelfSenderKeySyncStateChanged(
	broadcaster e2eePort.Broadcaster,
	userID shared.UserID,
	snapshot *SelfSenderKeySyncSnapshot,
) {
	if broadcaster == nil || snapshot == nil {
		return
	}
	payload, err := json.Marshal(struct {
		Type    string                     `json:"type"`
		Payload *SelfSenderKeySyncSnapshot `json:"payload"`
	}{
		Type:    "e2ee.self_sender_key_sync_state_changed",
		Payload: snapshot,
	})
	if err != nil {
		return
	}
	broadcaster.SendToUser(strconv.FormatInt(int64(userID), 10), payload)
}

func loadCurrentParticipant(
	ctx context.Context,
	repo participant.Repository,
	userID shared.UserID,
) (*participant.Participant, error) {
	return repo.FindByUserID(ctx, userID)
}

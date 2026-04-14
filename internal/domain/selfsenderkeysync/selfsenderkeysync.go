package selfsenderkeysync

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type SelfSenderKeySync struct {
	ID                ID
	ParticipantID     participant.ID
	RequesterDeviceID shared.DeviceID
	ProviderDeviceID  *shared.DeviceID
	Status            Status
	RequestedAt       time.Time
	ProviderClaimedAt *time.Time
	UploadedAt        *time.Time
	CompletedAt       *time.Time
	FailedAt          *time.Time
	LastError         *string
	UpdatedAt         time.Time
}

func (s *SelfSenderKeySync) IsActive() bool {
	return s != nil && (s.Status == StatusPendingProvider || s.Status == StatusSyncing || s.Status == StatusUploaded)
}

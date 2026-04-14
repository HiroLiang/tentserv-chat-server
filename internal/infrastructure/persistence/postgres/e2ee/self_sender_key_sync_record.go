package e2ee

import "time"

type SelfSenderKeySyncRecord struct {
	ID                int64      `db:"id"`
	ParticipantID     int64      `db:"participant_id"`
	RequesterDeviceID string     `db:"requester_device_id"`
	ProviderDeviceID  *string    `db:"provider_device_id"`
	Status            string     `db:"status"`
	RequestedAt       time.Time  `db:"requested_at"`
	ProviderClaimedAt *time.Time `db:"provider_claimed_at"`
	UploadedAt        *time.Time `db:"uploaded_at"`
	CompletedAt       *time.Time `db:"completed_at"`
	FailedAt          *time.Time `db:"failed_at"`
	LastError         *string    `db:"last_error"`
	UpdatedAt         time.Time  `db:"updated_at"`
}

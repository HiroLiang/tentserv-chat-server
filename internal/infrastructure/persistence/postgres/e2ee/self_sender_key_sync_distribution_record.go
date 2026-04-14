package e2ee

import "time"

type SelfSenderKeySyncDistributionRecord struct {
	ID                  int64      `db:"id"`
	SelfSenderKeySyncID int64      `db:"self_sender_key_sync_id"`
	ParticipantID       int64      `db:"participant_id"`
	RequesterDeviceID   string     `db:"requester_device_id"`
	ProviderDeviceID    string     `db:"provider_device_id"`
	SenderMemberID      int64      `db:"sender_member_id"`
	SenderDeviceID      string     `db:"sender_device_id"`
	SenderKeyVersion    int64      `db:"sender_key_version"`
	DistributionMessage []byte     `db:"distribution_message"`
	Status              string     `db:"status"`
	CreatedAt           time.Time  `db:"created_at"`
	ConsumedAt          *time.Time `db:"consumed_at"`
	FailedAt            *time.Time `db:"failed_at"`
}

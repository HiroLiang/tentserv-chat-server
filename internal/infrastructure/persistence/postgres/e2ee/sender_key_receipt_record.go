package e2ee

import "time"

type SenderKeyReceiptRecord struct {
	ID               int64     `db:"id"`
	SenderMemberID   int64     `db:"sender_member_id"`
	SenderDeviceID   string    `db:"sender_device_id"`
	ReceiverMemberID int64     `db:"receiver_member_id"`
	ReceiverDeviceID string    `db:"receiver_device_id"`
	SenderKeyVersion int64     `db:"sender_key_version"`
	Source           string    `db:"source"`
	UpdatedAt        time.Time `db:"updated_at"`
}

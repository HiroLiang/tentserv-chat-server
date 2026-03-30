package e2ee

import "time"

type SenderKeyDistributionRecord struct {
	ID               int64     `db:"id"`
	SenderMemberID   int64     `db:"sender_member_id"`
	ReceiverMemberID int64     `db:"receiver_member_id"`
	ChainID          int       `db:"chain_id"`
	DistributedAt    time.Time `db:"distributed_at"`
}

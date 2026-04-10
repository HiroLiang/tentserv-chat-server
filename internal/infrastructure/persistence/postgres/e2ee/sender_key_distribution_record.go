package e2ee

import "time"

type SenderKeyDistributionRecord struct {
	ID                  int64      `db:"id"`
	SenderMemberID      int64      `db:"sender_member_id"`
	ReceiverMemberID    int64      `db:"receiver_member_id"`
	SenderKeyVersion    int64      `db:"sender_key_version"`
	ChainID             int64      `db:"chain_id"`
	DistributionMessage []byte     `db:"distribution_message"`
	Status              string     `db:"status"`
	DistributedAt       time.Time  `db:"distributed_at"`
	ConsumedAt          *time.Time `db:"consumed_at"`
	FailedAt            *time.Time `db:"failed_at"`
}

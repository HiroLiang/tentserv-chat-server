package senderkeydistribution

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
)

// SenderKeyDistribution records that receiver_member_id has fetched
// sender_member_id's sender key at chain_id.
type SenderKeyDistribution struct {
	ID               ID
	SenderMemberID   chatmember.ID
	ReceiverMemberID chatmember.ID
	ChainID          int
	DistributedAt    time.Time
}

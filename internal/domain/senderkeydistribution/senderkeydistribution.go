package senderkeydistribution

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
)

// SenderKeyDistribution records the latest sender key distribution state for a sender/receiver pair.
// ChainID is a legacy mirror of SenderKeyVersion and must track the same 64-bit version number.
type SenderKeyDistribution struct {
	ID                  ID
	SenderMemberID      chatmember.ID
	ReceiverMemberID    chatmember.ID
	RoomID              int64
	SenderKeyVersion    int64
	DistributionMessage []byte
	Status              Status
	ChainID             int64
	DistributedAt       time.Time
	ConsumedAt          *time.Time
	FailedAt            *time.Time
}

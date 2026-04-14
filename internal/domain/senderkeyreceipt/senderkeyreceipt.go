package senderkeyreceipt

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type ID int64

type Source string

const (
	SourceDistribution Source = "distribution"
	SourceSelfSync     Source = "self_sync"
	SourceSenderKeys   Source = "sender_keys"
)

type SenderKeyReceipt struct {
	ID               ID
	SenderMemberID   chatmember.ID
	SenderDeviceID   shared.DeviceID
	ReceiverMemberID chatmember.ID
	ReceiverDeviceID shared.DeviceID
	SenderKeyVersion int64
	Source           Source
	UpdatedAt        time.Time
}

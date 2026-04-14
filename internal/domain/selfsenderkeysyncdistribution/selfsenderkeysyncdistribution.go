package selfsenderkeysyncdistribution

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type ID int64

type SelfSenderKeySyncDistribution struct {
	ID                  ID
	SelfSenderKeySyncID selfsenderkeysync.ID
	ParticipantID       participant.ID
	RequesterDeviceID   shared.DeviceID
	ProviderDeviceID    shared.DeviceID
	SenderMemberID      chatmember.ID
	SenderDeviceID      shared.DeviceID
	SenderKeyVersion    int64
	DistributionMessage []byte
	Status              Status
	CreatedAt           time.Time
	ConsumedAt          *time.Time
	FailedAt            *time.Time
}

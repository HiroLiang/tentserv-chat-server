package senderkeyrequest

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type SenderKeyRequest struct {
	ID                ID
	RequesterMemberID chatmember.ID
	RequesterDeviceID shared.DeviceID
	ProviderMemberID  chatmember.ID
	ProviderDeviceID  shared.DeviceID
	CreatedAt         time.Time
	FulfilledAt       *time.Time
}

package membersenderkey

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type MemberSenderKey struct {
	ID               ID
	ChatMemberID     chatmember.ID
	SenderDeviceID   shared.DeviceID
	SenderKeyVersion int64
	KeyFingerprint   *string
	ChainID          ChainID
	CreatedAt        time.Time
}

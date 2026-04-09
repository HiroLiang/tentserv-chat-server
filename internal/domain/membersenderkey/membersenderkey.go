package membersenderkey

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
)

type MemberSenderKey struct {
	ID               ID
	ChatMemberID     chatmember.ID
	SenderKeyVersion int64
	KeyFingerprint   *string
	ChainID          ChainID
	CreatedAt        time.Time
}

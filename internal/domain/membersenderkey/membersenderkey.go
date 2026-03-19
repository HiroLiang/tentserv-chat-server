package membersenderkey

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
)

type MemberSenderKey struct {
	ID                  ID
	ChatMemberID        chatmember.ID
	ChainID             ChainID
	SenderKeyPublic     SenderKeyPublic
	DistributionMessage []byte
	CreatedAt           time.Time
}

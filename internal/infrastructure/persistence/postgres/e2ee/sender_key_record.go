package e2ee

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
)

type SenderKeyRecord struct {
	ID                  membersenderkey.ID      `db:"id"`
	ChatMemberID        chatmember.ID           `db:"chat_member_id"`
	ChainID             membersenderkey.ChainID `db:"chain_id"`
	SenderKeyPublic     []byte                  `db:"sender_key_public"`
	DistributionMessage []byte                  `db:"distribution_message"`
	CreatedAt           time.Time               `db:"created_at"`
}

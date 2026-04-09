package e2ee

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
)

type SenderKeyRecord struct {
	ID               membersenderkey.ID      `db:"id"`
	ChatMemberID     chatmember.ID           `db:"chat_member_id"`
	ChainID          membersenderkey.ChainID `db:"chain_id"`
	SenderKeyVersion int64                   `db:"sender_key_version"`
	KeyFingerprint   *string                 `db:"key_fingerprint"`
	CreatedAt        time.Time               `db:"created_at"`
}

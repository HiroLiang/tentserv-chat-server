package membersenderkey

import "github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"

type ID int64

type ChainID int

type SenderKeyPublic [32]byte

// ChatMemberID is an alias for chatmember.ID used in this package context.
type ChatMemberID = chatmember.ID

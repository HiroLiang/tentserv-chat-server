package chatmessage

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type ChatMessage struct {
	ID               ID
	RoomID           chatroom.ID
	SenderID         chatmember.ID
	SenderDeviceID   shared.DeviceID
	SenderKeyVersion int64
	Content          string
	Type             MessageType
	ReplyToID        *ID
	IsEdited         bool
	IsDeleted        bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

package chat

import (
	"time"

	"github.com/HiroLiang/goat-server/internal/domain/chatmember"
	"github.com/HiroLiang/goat-server/internal/domain/chatmessage"
	"github.com/HiroLiang/goat-server/internal/domain/chatroom"
)

type ChatMessageRecord struct {
	ID        chatmessage.ID          `db:"id"`
	RoomID    chatroom.ID             `db:"room_id"`
	SenderID  chatmember.ID           `db:"sender_id"`
	Content   string                  `db:"content"`
	Type      chatmessage.MessageType `db:"message_type"`
	ReplyToID *chatmessage.ID         `db:"reply_to_id"`
	IsEdited  bool                    `db:"is_edited"`
	IsDeleted bool                    `db:"is_deleted"`
	CreatedAt time.Time               `db:"created_at"`
	UpdatedAt time.Time               `db:"updated_at"`
}

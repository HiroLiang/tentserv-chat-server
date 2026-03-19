package chat

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
)

type ChatRoomRecord struct {
	ID          chatroom.ID       `db:"id"`
	Name        *string           `db:"name"`
	Description *string           `db:"description"`
	AvatarName  *string           `db:"avatar_name"`
	Type        chatroom.RoomType `db:"type"`
	MaxMembers  int               `db:"max_members"`
	AllowAgent  bool              `db:"allow_agent"`
	IsDeleted   bool              `db:"is_deleted"`
	CreatedAt   time.Time         `db:"created_at"`
	UpdatedAt   time.Time         `db:"updated_at"`
}

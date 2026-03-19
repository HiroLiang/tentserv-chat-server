package chat

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
)

type ChatMemberRecord struct {
	ID            chatmember.ID   `db:"id"`
	RoomID        chatroom.ID     `db:"room_id"`
	ParticipantID participant.ID  `db:"participant_id"`
	Role          chatmember.Role `db:"role"`
	IsMuted       bool            `db:"is_muted"`
	IsDeleted     bool            `db:"is_delete"`
	LastReadAt    *time.Time      `db:"last_read_at"`
	JoinedAt      time.Time       `db:"joined_at"`
	UpdatedAt     time.Time       `db:"updated_at"`
	DeletedAt     *time.Time      `db:"deleted_at"`
}

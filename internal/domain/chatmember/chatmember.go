package chatmember

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
)

type ChatMember struct {
	ID            ID
	RoomID        chatroom.ID
	ParticipantID participant.ID
	Role          Role
	IsMuted       bool
	IsDeleted     bool
	LastReadAt    *time.Time
	JoinedAt      time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

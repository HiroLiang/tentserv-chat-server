package chat

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatinvitation"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
)

type ChatInvitationRecord struct {
	ID        chatinvitation.ID     `db:"id"`
	RoomID    chatroom.ID           `db:"room_id"`
	InviterID participant.ID        `db:"inviter_id"`
	InviteeID participant.ID        `db:"invitee_id"`
	Status    chatinvitation.Status `db:"status"`
	CreatedAt time.Time             `db:"created_at"`
	UpdatedAt time.Time             `db:"updated_at"`
}

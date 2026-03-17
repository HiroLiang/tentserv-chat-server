package chatinvitation

import (
	"time"

	"github.com/HiroLiang/goat-server/internal/domain/chatroom"
	"github.com/HiroLiang/goat-server/internal/domain/participant"
)

type ID int64

type Status string

const (
	Pending  Status = "pending"
	Accepted Status = "accepted"
	Rejected Status = "rejected"
)

type ChatInvitation struct {
	ID        ID
	RoomID    chatroom.ID
	InviterID participant.ID
	InviteeID participant.ID
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

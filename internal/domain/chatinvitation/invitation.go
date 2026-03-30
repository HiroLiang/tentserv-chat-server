package chatinvitation

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
)

type ID int64

type Status string

const (
	Pending  Status = "pending"
	Accepted Status = "accepted"
	Rejected Status = "rejected"
	Blocked  Status = "blocked"
)

type InvitationType string

const (
	// JoinRequest is created by a user who requests to join a room themselves
	// (inviter == invitee). Handled by ApproveJoinRequestUseCase.
	JoinRequest InvitationType = "join_request"
	// Invitation is created by an owner/member who invites another participant.
	// Handled by RespondToInvitationUseCase.
	Invitation InvitationType = "invitation"
)

type ChatInvitation struct {
	ID             ID
	RoomID         chatroom.ID
	InviterID      participant.ID
	InviteeID      participant.ID
	Status         Status
	InvitationType InvitationType
	ExpiresAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

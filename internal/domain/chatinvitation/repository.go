package chatinvitation

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/domain/chatroom"
	"github.com/HiroLiang/goat-server/internal/domain/participant"
)

type Repository interface {
	Create(ctx context.Context, inv *ChatInvitation) error
	FindByID(ctx context.Context, id ID) (*ChatInvitation, error)
	FindByRoomAndInvitee(ctx context.Context, roomID chatroom.ID, inviteeID participant.ID) (*ChatInvitation, error)
	FindByRoom(ctx context.Context, roomID chatroom.ID) ([]*ChatInvitation, error)
	UpdateStatus(ctx context.Context, id ID, status Status) error
}

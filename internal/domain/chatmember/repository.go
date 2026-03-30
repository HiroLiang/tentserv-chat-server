package chatmember

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
)

type Repository interface {
	FindByID(ctx context.Context, id ID) (*ChatMember, error)
	FindByRoomAndParticipant(ctx context.Context, roomID chatroom.ID, participantID participant.ID) (*ChatMember, error)
	FindByRoom(ctx context.Context, roomID chatroom.ID) ([]*ChatMember, error)
	FindByParticipant(ctx context.Context, participantID participant.ID) ([]*ChatMember, error)
	Add(ctx context.Context, member *ChatMember) error
	Update(ctx context.Context, member *ChatMember) error
	// SoftDelete marks a member as deleted without removing the row.
	SoftDelete(ctx context.Context, id ID) error
	// Remove permanently deletes the member row (hard delete).
	Remove(ctx context.Context, roomID chatroom.ID, participantID participant.ID) error
}

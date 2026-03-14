package chatmember

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/domain/chatroom"
	"github.com/HiroLiang/goat-server/internal/domain/participant"
)

type Repository interface {
	FindByID(ctx context.Context, id ID) (*ChatMember, error)
	FindByRoomAndParticipant(ctx context.Context, roomID chatroom.ID, participantID participant.ID) (*ChatMember, error)
	FindByRoom(ctx context.Context, roomID chatroom.ID) ([]*ChatMember, error)
	FindByParticipant(ctx context.Context, participantID participant.ID) ([]*ChatMember, error)
	Add(ctx context.Context, member *ChatMember) error
	Update(ctx context.Context, member *ChatMember) error
	Remove(ctx context.Context, roomID chatroom.ID, participantID participant.ID) error
}

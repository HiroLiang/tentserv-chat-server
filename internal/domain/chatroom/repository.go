package chatroom

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/domain/participant"
)

type Repository interface {
	FindByID(ctx context.Context, id ID) (*ChatRoom, error)
	Create(ctx context.Context, room *ChatRoom) error
	FindDirectByParticipants(ctx context.Context, p1ID, p2ID participant.ID) (*ChatRoom, error)
	Update(ctx context.Context, room *ChatRoom) error
	SoftDelete(ctx context.Context, id ID) error
}

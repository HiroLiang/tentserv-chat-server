package chatmessage

import (
	"context"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
)

type Repository interface {
	FindByID(ctx context.Context, id ID) (*ChatMessage, error)
	FindByRoom(ctx context.Context, roomID chatroom.ID, limit, offset uint64) ([]*ChatMessage, error)
	FindByRoomBefore(ctx context.Context, roomID chatroom.ID, beforeID ID, limit uint64) ([]*ChatMessage, error)
	FindLatestByRoom(ctx context.Context, roomID chatroom.ID) (*ChatMessage, error)
	CountByRoomAfter(ctx context.Context, roomID chatroom.ID, since time.Time) (int64, error)
	FindBySender(ctx context.Context, senderID chatmember.ID) ([]*ChatMessage, error)
	Create(ctx context.Context, msg *ChatMessage) error
	Update(ctx context.Context, msg *ChatMessage) error
	SoftDelete(ctx context.Context, id ID) error
}

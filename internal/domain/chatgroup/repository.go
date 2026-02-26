package chatgroup

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/domain/participant"
	"github.com/HiroLiang/goat-server/internal/domain/user"
)

type Repository interface {
	FindByID(ctx context.Context, id ID) (*ChatGroup, error)
	FindByCreator(ctx context.Context, creatorID user.ID) ([]*ChatGroup, error)
	FindDirectByParticipants(ctx context.Context, p1ID, p2ID participant.ID) (*ChatGroup, error)
	Create(ctx context.Context, group *ChatGroup) error
	Update(ctx context.Context, group *ChatGroup) error
	SoftDelete(ctx context.Context, id ID) error
}

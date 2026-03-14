package participant

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type Repository interface {
	FindByID(ctx context.Context, id ID) (*Participant, error)
	FindByUserID(ctx context.Context, userID shared.UserID) (*Participant, error)
	FindByAgentID(ctx context.Context, agentID int64) (*Participant, error)
	FindSystem(ctx context.Context) (*Participant, error)
	Create(ctx context.Context, p *Participant) error
}

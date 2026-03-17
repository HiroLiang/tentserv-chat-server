package agent

import "context"

type Repository interface {
	FindByID(ctx context.Context, id ID) (*Agent, error)
	FindAll(ctx context.Context) ([]*Agent, error)
	FindAllByStatus(ctx context.Context, status Status) ([]*Agent, error)
	Create(ctx context.Context, agent *Agent) error
}

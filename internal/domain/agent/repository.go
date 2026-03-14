package agent

import "context"

type Repository interface {
	FindAll(ctx context.Context) ([]*Agent, error)
	FindAllByStatus(ctx context.Context, status Status) ([]*Agent, error)
	Create(ctx context.Context, agent *Agent) error
}

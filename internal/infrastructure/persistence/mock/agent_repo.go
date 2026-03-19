package mock

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/agent"
)

type AgentRepository struct{}

var _ agent.Repository = (*AgentRepository)(nil)

func MockAgentRepository() *AgentRepository {
	return &AgentRepository{}
}

func (a AgentRepository) FindByID(ctx context.Context, id agent.ID) (*agent.Agent, error) {
	//TODO implement me
	panic("implement me")
}

func (a AgentRepository) FindAll(ctx context.Context) ([]*agent.Agent, error) {
	//TODO implement me
	panic("implement me")
}

func (a AgentRepository) FindAllByStatus(ctx context.Context, status agent.Status) ([]*agent.Agent, error) {
	//TODO implement me
	panic("implement me")
}

func (a AgentRepository) Create(ctx context.Context, agent *agent.Agent) error {
	//TODO implement me
	panic("implement me")
}

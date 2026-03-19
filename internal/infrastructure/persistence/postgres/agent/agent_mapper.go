package agent

import "github.com/HiroLiang/tentserv-chat-server/internal/domain/agent"

func toDomain(record *AgentRecord) (*agent.Agent, error) {
	return &agent.Agent{
		ID:        record.ID,
		Name:      record.Name,
		Type:      record.Type,
		Status:    record.Status,
		Engine:    record.Engine,
		CreatedAt: record.CreatedAt,
		CreatedBy: record.CreatedBy,
		UpdatedAt: record.UpdatedAt,
		UpdatedBy: record.UpdatedBy,
	}, nil
}

func toRecord(agent *agent.Agent) *AgentRecord {
	return &AgentRecord{
		ID:        agent.ID,
		Name:      agent.Name,
		Type:      agent.Type,
		Status:    agent.Status,
		Engine:    agent.Engine,
		CreatedAt: agent.CreatedAt,
		CreatedBy: agent.CreatedBy,
		UpdatedAt: agent.UpdatedAt,
		UpdatedBy: agent.UpdatedBy,
	}
}

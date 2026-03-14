package chat

import "github.com/HiroLiang/goat-server/internal/domain/participant"

func toParticipantDomain(rec *ParticipantRecord) (*participant.Participant, error) {
	return &participant.Participant{
		ID:         rec.ID,
		Type:       rec.Type,
		UserID:     rec.UserID,
		AgentID:    rec.AgentID,
		SystemType: rec.SystemType,
		CreatedAt:  rec.CreatedAt,
	}, nil
}

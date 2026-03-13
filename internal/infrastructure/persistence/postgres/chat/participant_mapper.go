package chat

import "github.com/HiroLiang/goat-server/internal/domain/participant"

func toParticipantDomain(rec *ParticipantRecord) (*participant.Participant, error) {
	return &participant.Participant{
		ID:          rec.ID,
		Type:        rec.Type,
		UserID:      rec.UserID,
		AgentID:     rec.AgentID,
		DisplayName: rec.DisplayName,
		AvatarName:  rec.AvatarName,
		CreatedAt:   rec.CreatedAt,
	}, nil
}

func toParticipantRecordRecord(p *participant.Participant) *ParticipantRecord {
	return &ParticipantRecord{
		ID:          p.ID,
		Type:        p.Type,
		UserID:      p.UserID,
		AgentID:     p.AgentID,
		DisplayName: p.DisplayName,
		AvatarName:  p.AvatarName,
		CreatedAt:   p.CreatedAt,
	}
}

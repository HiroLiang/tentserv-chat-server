package chat

import (
	"time"

	"github.com/HiroLiang/goat-server/internal/domain/participant"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type ParticipantRecord struct {
	ID         participant.ID              `db:"id"`
	Type       participant.ParticipantType `db:"type"`
	UserID     *shared.UserID              `db:"user_id"`
	AgentID    *int64                      `db:"agent_id"`
	SystemType *string                     `db:"system_type"`
	CreatedAt  time.Time                   `db:"created_at"`
}

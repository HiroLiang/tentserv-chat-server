package chat

import (
	"time"

	"github.com/HiroLiang/goat-server/internal/domain/agent"
	"github.com/HiroLiang/goat-server/internal/domain/participant"
	"github.com/HiroLiang/goat-server/internal/domain/user"
)

type ParticipantRecord struct {
	ID          participant.ID              `db:"id"`
	Type        participant.ParticipantType `db:"type"`
	UserID      *user.ID                    `db:"user_id"`
	AgentID     *agent.ID                   `db:"agent_id"`
	DisplayName string                      `db:"display_name"`
	AvatarName  string                      `db:"avatar_name"`
	CreatedAt   time.Time                   `db:"created_at"`
}

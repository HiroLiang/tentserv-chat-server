package participant

import (
	"time"

	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type Participant struct {
	ID         ID
	Type       ParticipantType
	UserID     *shared.UserID
	AgentID    *int64
	SystemType *string
	CreatedAt  time.Time
}

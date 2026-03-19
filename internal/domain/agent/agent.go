package agent

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type Agent struct {
	ID        ID
	Name      string
	Type      Type
	Status    Status
	Engine    Engine
	CreatedAt time.Time
	CreatedBy shared.UserID
	UpdatedAt time.Time
	UpdatedBy shared.UserID
}

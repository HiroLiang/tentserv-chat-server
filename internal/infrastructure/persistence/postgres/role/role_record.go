package role

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type RoleRecord struct {
	ID          shared.RoleID  `db:"id"`
	Code        role.Code      `db:"code"`
	Name        string         `db:"name"`
	Description *string        `db:"description"`
	CreatedBy   *shared.UserID `db:"created_by"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
}

package user

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

// ID is an alias for shared.UserID, used by other domain packages.
type ID = shared.UserID

type User struct {
	ID        shared.UserID
	AccountID shared.AccountID
	Name      string
	Avatar    string
	RoleCodes []role.Code
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewUser(accountID shared.AccountID, name string) *User {
	return &User{
		AccountID: accountID,
		Name:      name,
		RoleCodes: []role.Code{role.User},
	}
}

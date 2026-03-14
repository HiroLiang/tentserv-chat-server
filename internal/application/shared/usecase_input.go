package shared

import (
	"net"

	"github.com/HiroLiang/goat-server/internal/domain/auth"
	"github.com/HiroLiang/goat-server/internal/domain/role"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type RequestContext struct {
	IP       net.IP
	TraceID  string
	DeviceID shared.DeviceID
}

type AuthContext struct {
	AccountID   shared.AccountID
	UserID      shared.UserID
	Roles       []role.Code
	AccessToken auth.AccessToken
}

type BaseContext struct {
	Request RequestContext
	Auth    *AuthContext
}
type UseCaseInput[T any] struct {
	Base BaseContext
	Data T
}

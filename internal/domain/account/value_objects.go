package account

import (
	"net"
	"time"

	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type Status string

const (
	Active   Status = "active"
	Inactive Status = "inactive"
	Banned   Status = "banned"
	Applying Status = "applying"
	Deleted  Status = "deleted"
)

type AccountDevice struct {
	AccountID  shared.AccountID
	DeviceID   shared.DeviceID
	LastIP     net.IP
	LastSeenAt time.Time
}

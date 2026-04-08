package account

import (
	"net"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
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

type AccountLoginEvent struct {
	AccountID shared.AccountID
	DeviceID  shared.DeviceID
	IPAddress net.IP
	UserAgent string
	Success   bool
	CreatedAt time.Time
}

package useridentitykey

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type UserIdentityKey struct {
	ID            ID
	UserID        user.ID
	DeviceID      device.ID
	PublicKey     PublicKey
	SignPublicKey SignPublicKey
	Fingerprint   Fingerprint
	UploadedAt    time.Time
}

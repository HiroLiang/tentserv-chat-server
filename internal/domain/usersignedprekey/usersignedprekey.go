package usersignedprekey

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type UserSignedPreKey struct {
	ID        ID
	UserID    user.ID
	DeviceID  device.ID
	KeyID     KeyID
	PublicKey PublicKey
	Signature Signature
	IsActive  bool
	CreatedAt time.Time
	ExpiresAt *time.Time
}

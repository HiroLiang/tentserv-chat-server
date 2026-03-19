package e2ee

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/usersignedprekey"
)

type SignedPreKeyRecord struct {
	ID        usersignedprekey.ID    `db:"id"`
	UserID    user.ID                `db:"user_id"`
	DeviceID  device.ID              `db:"device_id"`
	KeyID     usersignedprekey.KeyID `db:"key_id"`
	PublicKey []byte                 `db:"public_key"`
	Signature []byte                 `db:"signature"`
	IsActive  bool                   `db:"is_active"`
	CreatedAt time.Time              `db:"created_at"`
	ExpiresAt *time.Time             `db:"expires_at"`
}

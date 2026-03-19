package e2ee

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/useridentitykey"
)

type IdentityKeyRecord struct {
	ID          useridentitykey.ID `db:"id"`
	UserID      user.ID            `db:"user_id"`
	DeviceID    device.ID          `db:"device_id"`
	PublicKey   []byte             `db:"public_key"`
	Fingerprint string             `db:"fingerprint"`
	UploadedAt  time.Time          `db:"uploaded_at"`
}

package e2ee

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userotpprekey"
)

type OTPPreKeyRecord struct {
	ID         userotpprekey.ID    `db:"id"`
	UserID     user.ID             `db:"user_id"`
	DeviceID   device.ID           `db:"device_id"`
	KeyID      userotpprekey.KeyID `db:"key_id"`
	PublicKey  []byte              `db:"public_key"`
	UploadedAt time.Time           `db:"uploaded_at"`
}

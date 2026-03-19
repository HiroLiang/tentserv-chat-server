package userotpprekey

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type UserOTPPreKey struct {
	ID         ID
	UserID     user.ID
	DeviceID   device.ID
	KeyID      KeyID
	PublicKey  PublicKey
	UploadedAt time.Time
}

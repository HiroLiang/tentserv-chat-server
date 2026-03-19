package userotpprekey

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type Repository interface {
	// ConsumeOne atomically deletes and returns one OTP prekey via DELETE ... RETURNING + FOR UPDATE SKIP LOCKED.
	ConsumeOne(ctx context.Context, userID user.ID, deviceID device.ID) (*UserOTPPreKey, error)
	AddBatch(ctx context.Context, keys []*UserOTPPreKey) error
	CountAvailable(ctx context.Context, userID user.ID, deviceID device.ID) (int, error)
}

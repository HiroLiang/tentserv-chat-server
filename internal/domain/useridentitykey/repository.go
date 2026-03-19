package useridentitykey

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type Repository interface {
	FindByUserAndDevice(ctx context.Context, userID user.ID, deviceID device.ID) (*UserIdentityKey, error)
	FindByUser(ctx context.Context, userID user.ID) ([]*UserIdentityKey, error)
	Upsert(ctx context.Context, key *UserIdentityKey) error
}

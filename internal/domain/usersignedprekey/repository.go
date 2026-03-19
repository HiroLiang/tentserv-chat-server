package usersignedprekey

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type Repository interface {
	FindActive(ctx context.Context, userID user.ID, deviceID device.ID) (*UserSignedPreKey, error)
	FindByKeyID(ctx context.Context, userID user.ID, deviceID device.ID, keyID KeyID) (*UserSignedPreKey, error)
	Add(ctx context.Context, key *UserSignedPreKey) error
	DeactivateAll(ctx context.Context, userID user.ID, deviceID device.ID) error
}

package friendship

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type Repository interface {
	FindByUserID(ctx context.Context, userID shared.UserID) ([]*Friendship, error)
}

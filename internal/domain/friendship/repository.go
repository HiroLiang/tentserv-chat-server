package friendship

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type Repository interface {
	FindByUserID(ctx context.Context, userID shared.UserID) ([]*Friendship, error)
	// FindAllByUserID returns all friendships where the user appears as either initiator or recipient.
	FindAllByUserID(ctx context.Context, userID shared.UserID) ([]*Friendship, error)
	// FindPendingByUserID returns pending friendships initiated by the user (sent requests).
	FindPendingByUserID(ctx context.Context, userID shared.UserID) ([]*Friendship, error)
	Create(ctx context.Context, userID, friendID shared.UserID) error
	CreateBlocked(ctx context.Context, userID, friendID shared.UserID) error
	FindByID(ctx context.Context, id int64) (*Friendship, error)
	FindByUserIDAndFriendID(ctx context.Context, userID, friendID shared.UserID) (*Friendship, error)
	FindBetweenUsers(ctx context.Context, userID1, userID2 shared.UserID) (*Friendship, error)
	FindPendingByFriendID(ctx context.Context, friendID shared.UserID) ([]*Friendship, error)
	UpdateStatus(ctx context.Context, id int64, status Status) error
	Delete(ctx context.Context, id int64) error
}

package friendship

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type Friendship struct {
	ID        int64
	UserID    shared.UserID
	FriendID  shared.UserID
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

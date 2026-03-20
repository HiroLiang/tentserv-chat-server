package friendship

import (
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
)

func toDomain(r *FriendshipRecord) *friendship.Friendship {
	return &friendship.Friendship{
		ID:        r.ID,
		UserID:    r.UserID,
		FriendID:  r.FriendID,
		Status:    friendship.Status(r.Status),
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

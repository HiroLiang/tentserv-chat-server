package usecase

import (
	"context"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type FriendItem struct {
	FriendshipID int64
	UserID       int64
	Name         string
	Avatar       string
	Status       string
	CreatedAt    time.Time
}

type GetFriendsOutput struct {
	Friends []FriendItem
}

type GetFriendsUseCase struct {
	friendshipRepo friendship.Repository
	userRepo       user.Repository
}

func NewGetFriendsUseCase(friendshipRepo friendship.Repository, userRepo user.Repository) *GetFriendsUseCase {
	return &GetFriendsUseCase{friendshipRepo: friendshipRepo, userRepo: userRepo}
}

func (uc *GetFriendsUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[struct{}],
) (*GetFriendsOutput, error) {
	friendships, err := uc.friendshipRepo.FindByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, err
	}

	items := make([]FriendItem, 0, len(friendships))
	for _, f := range friendships {
		u, err := uc.userRepo.FindByID(ctx, f.FriendID)
		if err != nil {
			continue
		}
		items = append(items, FriendItem{
			FriendshipID: f.ID,
			UserID:       int64(f.FriendID),
			Name:         u.Name,
			Avatar:       u.Avatar,
			Status:       string(f.Status),
			CreatedAt:    f.CreatedAt,
		})
	}
	return &GetFriendsOutput{Friends: items}, nil
}

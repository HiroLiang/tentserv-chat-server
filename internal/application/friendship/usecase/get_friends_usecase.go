package usecase

import (
	"context"
	"errors"
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
	BlockedBy    *string
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
		if f.Status != friendship.StatusAccepted && f.Status != friendship.StatusPending {
			continue
		}
		status := string(f.Status)
		var blockedBy *string
		reverse, reverseErr := uc.friendshipRepo.FindByUserIDAndFriendID(ctx, f.FriendID, f.UserID)
		if reverseErr != nil && !errors.Is(reverseErr, friendship.ErrFriendshipNotFound) {
			return nil, reverseErr
		}
		if reverseErr == nil && reverse.Status == friendship.StatusBlocked {
			status = string(friendship.StatusBlocked)
			v := BlockedByThem
			blockedBy = &v
		}
		u, err := uc.userRepo.FindByID(ctx, f.FriendID)
		if err != nil {
			continue
		}
		items = append(items, FriendItem{
			FriendshipID: f.ID,
			UserID:       int64(f.FriendID),
			Name:         u.Name,
			Avatar:       u.Avatar,
			Status:       status,
			BlockedBy:    blockedBy,
			CreatedAt:    f.CreatedAt,
		})
	}
	return &GetFriendsOutput{Friends: items}, nil
}

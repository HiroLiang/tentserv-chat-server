package usecase

import (
	"context"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type GetBlockedUsersOutput struct {
	Blocked []FriendItem
}

type GetBlockedUsersUseCase struct {
	friendshipRepo friendship.Repository
	userRepo       user.Repository
}

func NewGetBlockedUsersUseCase(friendshipRepo friendship.Repository, userRepo user.Repository) *GetBlockedUsersUseCase {
	return &GetBlockedUsersUseCase{friendshipRepo: friendshipRepo, userRepo: userRepo}
}

func (uc *GetBlockedUsersUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[struct{}],
) (*GetBlockedUsersOutput, error) {
	friendships, err := uc.friendshipRepo.FindByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, err
	}

	items := make([]FriendItem, 0, len(friendships))
	for _, f := range friendships {
		if f.Status != friendship.StatusBlocked {
			continue
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
			Status:       string(f.Status),
			CreatedAt:    f.CreatedAt,
		})
	}
	return &GetBlockedUsersOutput{Blocked: items}, nil
}

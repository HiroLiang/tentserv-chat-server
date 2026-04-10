package usecase

import (
	"context"
	"errors"

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

	allFriendships, err := uc.friendshipRepo.FindAllByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, err
	}

	currentRows := make(map[int64]*friendship.Friendship, len(allFriendships))
	for _, f := range allFriendships {
		if f.UserID == input.Base.Auth.UserID {
			currentRows[int64(f.FriendID)] = f
		}
	}

	items := make([]FriendItem, 0, len(friendships))
	seen := make(map[int64]struct{}, len(friendships))
	for _, f := range friendships {
		if f.Status != friendship.StatusBlocked {
			continue
		}
		u, err := uc.userRepo.FindByID(ctx, f.FriendID)
		if err != nil {
			continue
		}
		blockedBy := BlockedByMe
		items = append(items, FriendItem{
			FriendshipID: f.ID,
			UserID:       int64(f.FriendID),
			Name:         u.Name,
			Avatar:       u.Avatar,
			Status:       string(f.Status),
			BlockedBy:    &blockedBy,
			CreatedAt:    f.CreatedAt,
		})
		seen[int64(f.FriendID)] = struct{}{}
	}

	for _, f := range allFriendships {
		if f.FriendID != input.Base.Auth.UserID || f.Status != friendship.StatusBlocked {
			continue
		}
		otherUserID := int64(f.UserID)
		if _, ok := seen[otherUserID]; ok {
			continue
		}
		if current, ok := currentRows[otherUserID]; ok &&
			(current.Status == friendship.StatusAccepted || current.Status == friendship.StatusPending) {
			continue
		}
		u, err := uc.userRepo.FindByID(ctx, f.UserID)
		if err != nil {
			if errors.Is(err, user.ErrUserNotFound) {
				continue
			}
			return nil, err
		}
		blockedBy := BlockedByThem
		items = append(items, FriendItem{
			FriendshipID: f.ID,
			UserID:       otherUserID,
			Name:         u.Name,
			Avatar:       u.Avatar,
			Status:       string(f.Status),
			BlockedBy:    &blockedBy,
			CreatedAt:    f.CreatedAt,
		})
	}
	return &GetBlockedUsersOutput{Blocked: items}, nil
}

package usecase

import (
	"context"
	"errors"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type FriendRequestItem struct {
	FriendshipID int64
	UserID       int64
	Name         string
	Avatar       string
	CreatedAt    time.Time
}

type GetFriendRequestsOutput struct {
	Requests []FriendRequestItem
}

type GetFriendRequestsUseCase struct {
	friendshipRepo friendship.Repository
	userRepo       user.Repository
}

func NewGetFriendRequestsUseCase(friendshipRepo friendship.Repository, userRepo user.Repository) *GetFriendRequestsUseCase {
	return &GetFriendRequestsUseCase{friendshipRepo: friendshipRepo, userRepo: userRepo}
}

func (uc *GetFriendRequestsUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[struct{}],
) (*GetFriendRequestsOutput, error) {
	pendingList, err := uc.friendshipRepo.FindPendingByFriendID(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, err
	}

	items := make([]FriendRequestItem, 0, len(pendingList))
	for _, f := range pendingList {
		currentRow, rowErr := uc.friendshipRepo.FindByUserIDAndFriendID(ctx, input.Base.Auth.UserID, f.UserID)
		if rowErr != nil && !errors.Is(rowErr, friendship.ErrFriendshipNotFound) {
			return nil, rowErr
		}
		if rowErr == nil && currentRow.Status == friendship.StatusBlocked {
			continue
		}
		u, err := uc.userRepo.FindByID(ctx, f.UserID)
		if err != nil {
			continue
		}
		items = append(items, FriendRequestItem{
			FriendshipID: f.ID,
			UserID:       int64(f.UserID),
			Name:         u.Name,
			Avatar:       u.Avatar,
			CreatedAt:    f.CreatedAt,
		})
	}
	return &GetFriendRequestsOutput{Requests: items}, nil
}

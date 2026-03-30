package usecase

import (
	"context"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type SentRequestItem struct {
	FriendshipID int64
	UserID       int64
	Name         string
	Avatar       string
	CreatedAt    time.Time
}

type GetSentRequestsOutput struct {
	Requests []SentRequestItem
}

type GetSentRequestsUseCase struct {
	friendshipRepo friendship.Repository
	userRepo       user.Repository
}

func NewGetSentRequestsUseCase(friendshipRepo friendship.Repository, userRepo user.Repository) *GetSentRequestsUseCase {
	return &GetSentRequestsUseCase{friendshipRepo: friendshipRepo, userRepo: userRepo}
}

func (uc *GetSentRequestsUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[struct{}],
) (*GetSentRequestsOutput, error) {
	pending, err := uc.friendshipRepo.FindPendingByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, err
	}

	items := make([]SentRequestItem, 0, len(pending))
	for _, f := range pending {
		u, err := uc.userRepo.FindByID(ctx, f.FriendID)
		if err != nil {
			continue
		}
		items = append(items, SentRequestItem{
			FriendshipID: f.ID,
			UserID:       int64(f.FriendID),
			Name:         u.Name,
			Avatar:       u.Avatar,
			CreatedAt:    f.CreatedAt,
		})
	}
	return &GetSentRequestsOutput{Requests: items}, nil
}

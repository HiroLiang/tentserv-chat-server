package usecase

import (
	"context"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
)

type GetFriendsOutput struct {
	Friends []*friendship.Friendship
}

type GetFriendsUseCase struct {
	friendshipRepo friendship.Repository
}

func NewGetFriendsUseCase(friendshipRepo friendship.Repository) *GetFriendsUseCase {
	return &GetFriendsUseCase{friendshipRepo: friendshipRepo}
}

func (uc *GetFriendsUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[struct{}],
) (*GetFriendsOutput, error) {
	friends, err := uc.friendshipRepo.FindByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, err
	}
	return &GetFriendsOutput{Friends: friends}, nil
}

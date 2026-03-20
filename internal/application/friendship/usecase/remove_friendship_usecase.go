package usecase

import (
	"context"
	"errors"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
)

type RemoveFriendshipInput struct {
	FriendshipID int64
}

type RemoveFriendshipUseCase struct {
	friendshipRepo friendship.Repository
}

func NewRemoveFriendshipUseCase(friendshipRepo friendship.Repository) *RemoveFriendshipUseCase {
	return &RemoveFriendshipUseCase{friendshipRepo: friendshipRepo}
}

func (uc *RemoveFriendshipUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[RemoveFriendshipInput],
) error {
	currentUserID := input.Base.Auth.UserID

	f, err := uc.friendshipRepo.FindByID(ctx, input.Data.FriendshipID)
	if err != nil {
		if errors.Is(err, friendship.ErrFriendshipNotFound) {
			return friendship.ErrFriendshipNotFound
		}
		return err
	}

	if f.UserID != currentUserID && f.FriendID != currentUserID {
		return friendship.ErrForbidden
	}

	return uc.friendshipRepo.Delete(ctx, f.ID)
}

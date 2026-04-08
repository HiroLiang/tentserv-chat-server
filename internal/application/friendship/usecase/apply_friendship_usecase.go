package usecase

import (
	"context"
	"errors"
	"strings"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type ApplyFriendshipInput struct {
	FriendID int64 `json:"friend_id"`
}

type ApplyFriendshipUseCase struct {
	friendshipRepo friendship.Repository
}

func NewApplyFriendshipUseCase(friendshipRepo friendship.Repository) *ApplyFriendshipUseCase {
	return &ApplyFriendshipUseCase{friendshipRepo: friendshipRepo}
}

func (uc *ApplyFriendshipUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[ApplyFriendshipInput],
) error {
	currentUserID := input.Base.Auth.UserID
	friendID := shared.UserID(input.Data.FriendID)
	if friendID == 0 || currentUserID == friendID {
		return friendship.ErrSelfFriendship
	}

	_, err := uc.friendshipRepo.FindByUserIDAndFriendID(ctx, currentUserID, friendID)
	if err == nil {
		return friendship.ErrFriendshipAlreadyExists
	}
	if !errors.Is(err, friendship.ErrFriendshipNotFound) {
		return err
	}

	_, err = uc.friendshipRepo.FindByUserIDAndFriendID(ctx, friendID, currentUserID)
	if err == nil {
		return friendship.ErrFriendshipAlreadyExists
	}
	if !errors.Is(err, friendship.ErrFriendshipNotFound) {
		return err
	}

	// Catch DB unique violation from concurrent requests and surface a friendly error
	if err := uc.friendshipRepo.Create(ctx, currentUserID, friendID); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key value") {
			return friendship.ErrFriendshipAlreadyExists
		}
		return err
	}
	return nil
}

package usecase

import (
	"context"
	"errors"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type UnblockUserInput struct {
	TargetUserID int64
}

type UnblockUserUseCase struct {
	friendshipRepo friendship.Repository
}

func NewUnblockUserUseCase(friendshipRepo friendship.Repository) *UnblockUserUseCase {
	return &UnblockUserUseCase{friendshipRepo: friendshipRepo}
}

func (uc *UnblockUserUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[UnblockUserInput],
) error {
	currentUserID := input.Base.Auth.UserID
	targetID := shared.UserID(input.Data.TargetUserID)

	existing, err := uc.friendshipRepo.FindByUserIDAndFriendID(ctx, currentUserID, targetID)
	if err != nil {
		if errors.Is(err, friendship.ErrFriendshipNotFound) {
			return friendship.ErrNotBlocked
		}
		return err
	}

	if existing.Status != friendship.StatusBlocked {
		return friendship.ErrNotBlocked
	}

	return uc.friendshipRepo.Delete(ctx, existing.ID)
}

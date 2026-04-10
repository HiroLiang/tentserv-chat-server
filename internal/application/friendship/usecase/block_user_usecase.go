package usecase

import (
	"context"
	"errors"
	"strings"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type BlockUserInput struct {
	TargetUserID int64
}

type BlockUserUseCase struct {
	friendshipRepo friendship.Repository
}

func NewBlockUserUseCase(friendshipRepo friendship.Repository) *BlockUserUseCase {
	return &BlockUserUseCase{friendshipRepo: friendshipRepo}
}

func (uc *BlockUserUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[BlockUserInput],
) error {
	currentUserID := input.Base.Auth.UserID
	targetID := shared.UserID(input.Data.TargetUserID)

	// Check if blocker→target record exists
	existing, err := uc.friendshipRepo.FindByUserIDAndFriendID(ctx, currentUserID, targetID)
	if err != nil && !errors.Is(err, friendship.ErrFriendshipNotFound) {
		return err
	}

	if err == nil {
		if existing.Status == friendship.StatusBlocked {
			return friendship.ErrAlreadyBlocked
		}
		// Update existing record to blocked
		if updateErr := uc.friendshipRepo.UpdateStatus(ctx, existing.ID, friendship.StatusBlocked); updateErr != nil {
			return updateErr
		}
	} else {
		// No existing record — create one with blocked status
		if createErr := uc.friendshipRepo.CreateBlocked(ctx, currentUserID, targetID); createErr != nil {
			if strings.Contains(strings.ToLower(createErr.Error()), "duplicate key value") {
				return friendship.ErrAlreadyBlocked
			}
			return createErr
		}
	}

	return nil
}

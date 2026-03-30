package usecase

import (
	"context"
	"errors"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
)

type CancelSentRequestInput struct {
	FriendshipID int64
}

type CancelSentRequestUseCase struct {
	friendshipRepo friendship.Repository
}

func NewCancelSentRequestUseCase(friendshipRepo friendship.Repository) *CancelSentRequestUseCase {
	return &CancelSentRequestUseCase{friendshipRepo: friendshipRepo}
}

func (uc *CancelSentRequestUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[CancelSentRequestInput],
) error {
	currentUserID := input.Base.Auth.UserID

	f, err := uc.friendshipRepo.FindByID(ctx, input.Data.FriendshipID)
	if err != nil {
		if errors.Is(err, friendship.ErrFriendshipNotFound) {
			return friendship.ErrFriendshipNotFound
		}
		return err
	}

	// Only the initiator can cancel
	if f.UserID != currentUserID {
		return friendship.ErrForbidden
	}

	// Only pending requests can be cancelled
	if f.Status != friendship.StatusPending {
		return friendship.ErrFriendshipNotPending
	}

	return uc.friendshipRepo.Delete(ctx, f.ID)
}

package usecase

import (
	"context"
	"errors"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
)

type AcceptFriendshipInput struct {
	FriendshipID int64
}

type AcceptFriendshipUseCase struct {
	uow            transaction.UnitOfWork
	friendshipRepo friendship.Repository
}

func NewAcceptFriendshipUseCase(uow transaction.UnitOfWork, friendshipRepo friendship.Repository) *AcceptFriendshipUseCase {
	return &AcceptFriendshipUseCase{uow: uow, friendshipRepo: friendshipRepo}
}

func (uc *AcceptFriendshipUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[AcceptFriendshipInput],
) error {
	currentUserID := input.Base.Auth.UserID

	f, err := uc.friendshipRepo.FindByID(ctx, input.Data.FriendshipID)
	if err != nil {
		if errors.Is(err, friendship.ErrFriendshipNotFound) {
			return friendship.ErrFriendshipNotFound
		}
		return err
	}

	if f.FriendID != currentUserID {
		return friendship.ErrForbidden
	}

	if f.Status != friendship.StatusPending {
		return friendship.ErrFriendshipNotPending
	}

	ctx, tx, err := uc.uow.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := uc.friendshipRepo.UpdateStatus(ctx, f.ID, friendship.StatusAccepted); err != nil {
		return err
	}

	if err := uc.friendshipRepo.Create(ctx, currentUserID, f.UserID); err != nil {
		return err
	}

	// Update the reverse row status to accepted immediately
	reverse, err := uc.friendshipRepo.FindByUserIDAndFriendID(ctx, currentUserID, f.UserID)
	if err == nil {
		_ = uc.friendshipRepo.UpdateStatus(ctx, reverse.ID, friendship.StatusAccepted)
	}

	return tx.Commit()
}

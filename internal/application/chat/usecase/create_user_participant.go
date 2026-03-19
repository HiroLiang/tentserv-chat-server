package usecase

import (
	"context"
	"errors"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
)

type CreateUserParticipantOutput struct {
	ID        int64
	UserID    int64
	CreatedAt time.Time
}

type CreateUserParticipantUseCase struct {
	uow             transaction.UnitOfWork
	participantRepo participant.Repository
}

func NewCreateUserParticipantUseCase(
	uow transaction.UnitOfWork,
	participantRepo participant.Repository,
) *CreateUserParticipantUseCase {
	return &CreateUserParticipantUseCase{
		uow:             uow,
		participantRepo: participantRepo,
	}
}

func (uc *CreateUserParticipantUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[struct{}],
) (CreateUserParticipantOutput, error) {
	userID := input.Base.Auth.UserID

	ctx, tx, err := uc.uow.Begin(ctx)
	if err != nil {
		return CreateUserParticipantOutput{}, ErrCreateParticipant
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = uc.participantRepo.FindByUserID(ctx, userID)
	if err == nil {
		return CreateUserParticipantOutput{}, ErrParticipantAlreadyExists
	}
	if !errors.Is(err, participant.ErrNotFound) {
		return CreateUserParticipantOutput{}, ErrCreateParticipant
	}

	p := participant.Participant{
		Type:   participant.UserType,
		UserID: &userID,
	}

	if err := uc.participantRepo.Create(ctx, &p); err != nil {
		return CreateUserParticipantOutput{}, ErrCreateParticipant
	}

	if err := tx.Commit(); err != nil {
		return CreateUserParticipantOutput{}, ErrCreateParticipant
	}

	return CreateUserParticipantOutput{
		ID:        int64(p.ID),
		UserID:    int64(userID),
		CreatedAt: p.CreatedAt,
	}, nil
}

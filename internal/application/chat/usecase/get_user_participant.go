package usecase

import (
	"context"
	"errors"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
)

type GetUserParticipantOutput struct {
	ID        int64
	UserID    int64
	CreatedAt time.Time
}

type GetUserParticipantUseCase struct {
	participantRepo participant.Repository
}

func NewGetUserParticipantUseCase(participantRepo participant.Repository) *GetUserParticipantUseCase {
	return &GetUserParticipantUseCase{
		participantRepo: participantRepo,
	}
}

func (uc *GetUserParticipantUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[struct{}],
) (GetUserParticipantOutput, error) {
	userID := input.Base.Auth.UserID

	p, err := uc.participantRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return GetUserParticipantOutput{}, ErrParticipantNotFound
		}
		return GetUserParticipantOutput{}, err
	}

	return GetUserParticipantOutput{
		ID:        int64(p.ID),
		UserID:    int64(userID),
		CreatedAt: p.CreatedAt,
	}, nil
}

package usecase

import (
	"context"
	"errors"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
)

type GetSelfSenderKeySyncInput struct{}

type GetSelfSenderKeySyncOutput = SelfSenderKeySyncSnapshot

type GetSelfSenderKeySyncUseCase struct {
	participantRepo participant.Repository
	selfSyncRepo    selfsenderkeysync.Repository
	accountRepo     account.Repository
	deviceRepo      device.Repository
}

func NewGetSelfSenderKeySyncUseCase(
	participantRepo participant.Repository,
	selfSyncRepo selfsenderkeysync.Repository,
	accountRepo account.Repository,
	deviceRepo device.Repository,
) *GetSelfSenderKeySyncUseCase {
	return &GetSelfSenderKeySyncUseCase{
		participantRepo: participantRepo,
		selfSyncRepo:    selfSyncRepo,
		accountRepo:     accountRepo,
		deviceRepo:      deviceRepo,
	}
}

func (u *GetSelfSenderKeySyncUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[GetSelfSenderKeySyncInput],
) (*GetSelfSenderKeySyncOutput, error) {
	currentParticipant, err := loadCurrentParticipant(ctx, u.participantRepo, input.Base.Auth.UserID)
	if err != nil {
		return nil, ErrNotRoomMember
	}
	syncState, err := u.selfSyncRepo.FindByParticipantID(ctx, currentParticipant.ID)
	if err != nil {
		if !errors.Is(err, selfsenderkeysync.ErrNotFound) {
			return nil, ErrNotRoomMember
		}
		syncState = nil
	}
	return buildSelfSenderKeySyncSnapshot(ctx, u.accountRepo, u.deviceRepo, input.Base.Auth.AccountID, input.Base.Request.DeviceID, syncState)
}

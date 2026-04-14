package usecase

import (
	"context"
	"encoding/base64"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysyncdistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type PendingSelfSenderKeySyncDistributionItem struct {
	DistributionID      int64
	SenderMemberID      int64
	SenderDeviceID      string
	SenderKeyVersion    int64
	DistributionMessage string
}

type GetPendingSelfSenderKeySyncDistributionsInput struct{}

type GetPendingSelfSenderKeySyncDistributionsOutput struct {
	Distributions []PendingSelfSenderKeySyncDistributionItem
}

type GetPendingSelfSenderKeySyncDistributionsUseCase struct {
	participantRepo participant.Repository
	selfSyncRepo    selfsenderkeysync.Repository
	copyRepo        selfsenderkeysyncdistribution.Repository
}

func NewGetPendingSelfSenderKeySyncDistributionsUseCase(
	participantRepo participant.Repository,
	selfSyncRepo selfsenderkeysync.Repository,
	copyRepo selfsenderkeysyncdistribution.Repository,
) *GetPendingSelfSenderKeySyncDistributionsUseCase {
	return &GetPendingSelfSenderKeySyncDistributionsUseCase{
		participantRepo: participantRepo,
		selfSyncRepo:    selfSyncRepo,
		copyRepo:        copyRepo,
	}
}

func (u *GetPendingSelfSenderKeySyncDistributionsUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[GetPendingSelfSenderKeySyncDistributionsInput],
) (*GetPendingSelfSenderKeySyncDistributionsOutput, error) {
	currentParticipant, syncState, err := u.loadRequesterSync(ctx, input.Base.Auth.UserID, input.Base.Request.DeviceID)
	if err != nil {
		return nil, err
	}

	items, err := u.copyRepo.FindPendingByRequester(ctx, syncState.ID, currentParticipant.ID, input.Base.Request.DeviceID)
	if err != nil {
		return nil, err
	}

	out := &GetPendingSelfSenderKeySyncDistributionsOutput{
		Distributions: make([]PendingSelfSenderKeySyncDistributionItem, 0, len(items)),
	}
	for _, item := range items {
		out.Distributions = append(out.Distributions, PendingSelfSenderKeySyncDistributionItem{
			DistributionID:      int64(item.ID),
			SenderMemberID:      int64(item.SenderMemberID),
			SenderDeviceID:      item.SenderDeviceID.String(),
			SenderKeyVersion:    item.SenderKeyVersion,
			DistributionMessage: encodeDistributionMessage(item.DistributionMessage),
		})
	}
	return out, nil
}

func encodeDistributionMessage(raw []byte) string {
	return base64.StdEncoding.EncodeToString(raw)
}

func (u *GetPendingSelfSenderKeySyncDistributionsUseCase) loadRequesterSync(
	ctx context.Context,
	userID shared.UserID,
	currentDeviceID shared.DeviceID,
) (*participant.Participant, *selfsenderkeysync.SelfSenderKeySync, error) {
	currentParticipant, err := loadCurrentParticipant(ctx, u.participantRepo, userID)
	if err != nil {
		return nil, nil, ErrForbidden
	}
	syncState, err := u.selfSyncRepo.FindByParticipantID(ctx, currentParticipant.ID)
	if err != nil {
		return nil, nil, ErrForbidden
	}
	if syncState.RequesterDeviceID != currentDeviceID {
		return nil, nil, ErrForbidden
	}
	return currentParticipant, syncState, nil
}

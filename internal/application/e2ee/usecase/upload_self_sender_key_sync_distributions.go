package usecase

import (
	"context"
	"encoding/base64"
	"fmt"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysyncdistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type SelfSenderKeySyncDistributionUploadItem struct {
	SenderMemberID      int64
	SenderDeviceID      string
	SenderKeyVersion    int64
	DistributionMessage string
}

type BulkSelfSenderKeySyncDistributionsInput struct {
	Items []SelfSenderKeySyncDistributionUploadItem
}

type BulkSelfSenderKeySyncDistributionsOutput struct {
	Count int
}

type UploadSelfSenderKeySyncDistributionsUseCase struct {
	participantRepo participant.Repository
	selfSyncRepo    selfsenderkeysync.Repository
	copyRepo        selfsenderkeysyncdistribution.Repository
}

func NewUploadSelfSenderKeySyncDistributionsUseCase(
	participantRepo participant.Repository,
	selfSyncRepo selfsenderkeysync.Repository,
	copyRepo selfsenderkeysyncdistribution.Repository,
) *UploadSelfSenderKeySyncDistributionsUseCase {
	return &UploadSelfSenderKeySyncDistributionsUseCase{
		participantRepo: participantRepo,
		selfSyncRepo:    selfSyncRepo,
		copyRepo:        copyRepo,
	}
}

func (u *UploadSelfSenderKeySyncDistributionsUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[BulkSelfSenderKeySyncDistributionsInput],
) (*BulkSelfSenderKeySyncDistributionsOutput, error) {
	currentParticipant, syncState, err := u.loadProviderSync(ctx, input.Base.Auth.UserID, input.Base.Request.DeviceID)
	if err != nil {
		return nil, err
	}

	rows := make([]*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution, 0, len(input.Data.Items))
	for _, item := range input.Data.Items {
		senderDeviceID, parseErr := shared.ParseDeviceID(item.SenderDeviceID)
		if parseErr != nil {
			return nil, fmt.Errorf("%w: sender device id", ErrInvalidSignature)
		}
		if senderDeviceID == syncState.RequesterDeviceID {
			return nil, ErrForbidden
		}
		distBytes, decodeErr := base64.StdEncoding.DecodeString(item.DistributionMessage)
		if decodeErr != nil || len(distBytes) == 0 {
			return nil, fmt.Errorf("%w: decode distribution message", ErrInvalidSignature)
		}
		rows = append(rows, &selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution{
			SelfSenderKeySyncID: syncState.ID,
			ParticipantID:       currentParticipant.ID,
			RequesterDeviceID:   syncState.RequesterDeviceID,
			ProviderDeviceID:    input.Base.Request.DeviceID,
			SenderMemberID:      chatmember.ID(item.SenderMemberID),
			SenderDeviceID:      senderDeviceID,
			SenderKeyVersion:    item.SenderKeyVersion,
			DistributionMessage: distBytes,
			Status:              selfsenderkeysyncdistribution.StatusAvailable,
		})
	}

	if err := u.copyRepo.ReplaceForSync(ctx, syncState.ID, currentParticipant.ID, syncState.RequesterDeviceID, input.Base.Request.DeviceID, rows); err != nil {
		return nil, fmt.Errorf("replace self sender key sync distributions: %w", err)
	}

	return &BulkSelfSenderKeySyncDistributionsOutput{Count: len(rows)}, nil
}

func (u *UploadSelfSenderKeySyncDistributionsUseCase) loadProviderSync(
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
	if syncState.ProviderDeviceID == nil || *syncState.ProviderDeviceID != currentDeviceID {
		return nil, nil, ErrForbidden
	}
	if syncState.Status != selfsenderkeysync.StatusSyncing && syncState.Status != selfsenderkeysync.StatusUploaded {
		return nil, nil, ErrForbidden
	}
	return currentParticipant, syncState, nil
}

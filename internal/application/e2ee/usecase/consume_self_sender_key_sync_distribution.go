package usecase

import (
	"context"
	"fmt"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyreceipt"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysyncdistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type ConsumeSelfSenderKeySyncDistributionInput struct {
	DistributionID int64
	Status         string
}

type ConsumeSelfSenderKeySyncDistributionOutput struct{}

type ConsumeSelfSenderKeySyncDistributionUseCase struct {
	participantRepo participant.Repository
	chatMemberRepo  chatmember.Repository
	selfSyncRepo    selfsenderkeysync.Repository
	copyRepo        selfsenderkeysyncdistribution.Repository
	receiptRepo     senderkeyreceipt.Repository
}

func NewConsumeSelfSenderKeySyncDistributionUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	selfSyncRepo selfsenderkeysync.Repository,
	copyRepo selfsenderkeysyncdistribution.Repository,
	receiptRepo senderkeyreceipt.Repository,
) *ConsumeSelfSenderKeySyncDistributionUseCase {
	return &ConsumeSelfSenderKeySyncDistributionUseCase{
		participantRepo: participantRepo,
		chatMemberRepo:  chatMemberRepo,
		selfSyncRepo:    selfSyncRepo,
		copyRepo:        copyRepo,
		receiptRepo:     receiptRepo,
	}
}

func (u *ConsumeSelfSenderKeySyncDistributionUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[ConsumeSelfSenderKeySyncDistributionInput],
) (*ConsumeSelfSenderKeySyncDistributionOutput, error) {
	currentParticipant, syncState, err := u.loadRequesterSync(ctx, input.Base.Auth.UserID, input.Base.Request.DeviceID)
	if err != nil {
		return nil, err
	}

	distributionID := selfsenderkeysyncdistribution.ID(input.Data.DistributionID)
	switch input.Data.Status {
	case "consumed":
		dist, err := u.loadDistribution(ctx, syncState.ID, distributionID, currentParticipant.ID, input.Base.Request.DeviceID)
		if err != nil {
			return nil, err
		}
		if err := u.copyRepo.MarkConsumed(ctx, syncState.ID, distributionID, currentParticipant.ID, input.Base.Request.DeviceID); err != nil {
			return nil, err
		}
		if err := u.recordReceipt(ctx, currentParticipant.ID, input.Base.Request.DeviceID, dist); err != nil {
			return nil, err
		}
	case "failed":
		if err := u.copyRepo.MarkFailed(ctx, syncState.ID, distributionID, currentParticipant.ID, input.Base.Request.DeviceID); err != nil {
			return nil, err
		}
	default:
		return nil, ErrInvalidSignature
	}

	return &ConsumeSelfSenderKeySyncDistributionOutput{}, nil
}

func (u *ConsumeSelfSenderKeySyncDistributionUseCase) loadRequesterSync(
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

func (u *ConsumeSelfSenderKeySyncDistributionUseCase) loadDistribution(
	ctx context.Context,
	selfSyncID selfsenderkeysync.ID,
	distributionID selfsenderkeysyncdistribution.ID,
	participantID participant.ID,
	requesterDeviceID shared.DeviceID,
) (*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution, error) {
	items, err := u.copyRepo.FindPendingByRequester(ctx, selfSyncID, participantID, requesterDeviceID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.ID == distributionID {
			return item, nil
		}
	}
	return nil, fmt.Errorf("find self sender key sync distribution: %w", selfsenderkeysyncdistribution.ErrNotFound)
}

func (u *ConsumeSelfSenderKeySyncDistributionUseCase) recordReceipt(
	ctx context.Context,
	participantID participant.ID,
	requesterDeviceID shared.DeviceID,
	dist *selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution,
) error {
	senderMember, err := u.chatMemberRepo.FindByID(ctx, dist.SenderMemberID)
	if err != nil || senderMember.IsDeleted {
		return fmt.Errorf("find sender member for receipt: %w", err)
	}
	receiverMember, err := u.chatMemberRepo.FindByRoomAndParticipant(ctx, chatroom.ID(senderMember.RoomID), participantID)
	if err != nil || receiverMember.IsDeleted {
		return fmt.Errorf("find receiver member for receipt: %w", err)
	}
	return u.receiptRepo.Upsert(ctx, &senderkeyreceipt.SenderKeyReceipt{
		SenderMemberID:   dist.SenderMemberID,
		SenderDeviceID:   dist.SenderDeviceID,
		ReceiverMemberID: receiverMember.ID,
		ReceiverDeviceID: requesterDeviceID,
		SenderKeyVersion: dist.SenderKeyVersion,
		Source:           senderkeyreceipt.SourceSelfSync,
	})
}

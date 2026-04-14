package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	e2eePort "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyreceipt"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyrequest"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type ConsumeSenderKeyDistributionInput struct {
	DistributionID int64
	Status         string
}

type ConsumeSenderKeyDistributionOutput struct{}

type ConsumeSenderKeyDistributionUseCase struct {
	participantRepo      participant.Repository
	chatMemberRepo       chatmember.Repository
	distributionRepo     senderkeydistribution.Repository
	receiptRepo          senderkeyreceipt.Repository
	senderKeyRequestRepo senderkeyrequest.Repository
	broadcaster          e2eePort.Broadcaster
}

func NewConsumeSenderKeyDistributionUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	distributionRepo senderkeydistribution.Repository,
	receiptRepo senderkeyreceipt.Repository,
	senderKeyRequestRepo senderkeyrequest.Repository,
	broadcaster e2eePort.Broadcaster,
) *ConsumeSenderKeyDistributionUseCase {
	return &ConsumeSenderKeyDistributionUseCase{
		participantRepo:      participantRepo,
		chatMemberRepo:       chatMemberRepo,
		distributionRepo:     distributionRepo,
		receiptRepo:          receiptRepo,
		senderKeyRequestRepo: senderKeyRequestRepo,
		broadcaster:          broadcaster,
	}
}

func (u *ConsumeSenderKeyDistributionUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[ConsumeSenderKeyDistributionInput],
) (*ConsumeSenderKeyDistributionOutput, error) {
	callerParticipant, err := u.participantRepo.FindByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, ErrNotRoomMember
	}

	dist, err := u.distributionRepo.FindByID(ctx, senderkeydistribution.ID(input.Data.DistributionID))
	if err != nil {
		return nil, fmt.Errorf("find sender key distribution: %w", err)
	}

	receiverMember, err := u.chatMemberRepo.FindByID(ctx, dist.ReceiverMemberID)
	if err != nil || receiverMember.IsDeleted || receiverMember.ParticipantID != callerParticipant.ID {
		return nil, ErrNotRoomMember
	}
	if dist.ReceiverDeviceID != input.Base.Request.DeviceID {
		return nil, ErrNotRoomMember
	}

	switch input.Data.Status {
	case string(senderkeydistribution.StatusConsumed):
		if err := u.distributionRepo.MarkConsumed(ctx, dist.ID); err != nil {
			return nil, fmt.Errorf("mark sender key distribution consumed: %w", err)
		}
		if err := u.receiptRepo.Upsert(ctx, &senderkeyreceipt.SenderKeyReceipt{
			SenderMemberID:   dist.SenderMemberID,
			SenderDeviceID:   dist.SenderDeviceID,
			ReceiverMemberID: dist.ReceiverMemberID,
			ReceiverDeviceID: dist.ReceiverDeviceID,
			SenderKeyVersion: dist.SenderKeyVersion,
			Source:           senderkeyreceipt.SourceDistribution,
		}); err != nil {
			return nil, fmt.Errorf("record sender key receipt: %w", err)
		}
	case string(senderkeydistribution.StatusFailed):
		if err := u.distributionRepo.MarkFailed(ctx, dist.ID); err != nil {
			return nil, fmt.Errorf("mark sender key distribution failed: %w", err)
		}
		req := &senderkeyrequest.SenderKeyRequest{
			RequesterMemberID: receiverMember.ID,
			RequesterDeviceID: input.Base.Request.DeviceID,
			ProviderMemberID:  dist.SenderMemberID,
			ProviderDeviceID:  dist.SenderDeviceID,
		}
		if err := u.senderKeyRequestRepo.Upsert(ctx, req); err != nil {
			return nil, fmt.Errorf("requeue sender key request: %w", err)
		}
		go u.notifyProvider(context.Background(), dist.SenderMemberID, dist.SenderDeviceID, receiverMember, input.Base.Request.DeviceID)
	default:
		return nil, fmt.Errorf("%w: unsupported consume status", ErrInvalidSignature)
	}

	return &ConsumeSenderKeyDistributionOutput{}, nil
}

func (u *ConsumeSenderKeyDistributionUseCase) notifyProvider(
	ctx context.Context,
	providerMemberID chatmember.ID,
	providerDeviceID shared.DeviceID,
	requesterMember *chatmember.ChatMember,
	requesterDeviceID shared.DeviceID,
) {
	providerMember, err := u.chatMemberRepo.FindByID(ctx, providerMemberID)
	if err != nil || providerMember.IsDeleted {
		return
	}
	providerParticipant, err := u.participantRepo.FindByID(ctx, providerMember.ParticipantID)
	if err != nil || providerParticipant.UserID == nil {
		return
	}
	requesterParticipant, err := u.participantRepo.FindByID(ctx, requesterMember.ParticipantID)
	if err != nil || requesterParticipant.UserID == nil {
		return
	}

	payload, err := json.Marshal(struct {
		Type    string                   `json:"type"`
		Payload wsSenderKeyNeededPayload `json:"payload"`
	}{
		Type: "e2ee.sender_key_needed",
		Payload: wsSenderKeyNeededPayload{
			RoomID:            int64(requesterMember.RoomID),
			ProviderMemberID:  int64(providerMember.ID),
			ProviderDeviceID:  providerDeviceID.String(),
			RequesterMemberID: int64(requesterMember.ID),
			RequesterUserID:   int64(*requesterParticipant.UserID),
			RequesterDeviceID: requesterDeviceID.String(),
		},
	})
	if err != nil {
		return
	}

	providerUserIDStr := strconv.FormatInt(int64(*providerParticipant.UserID), 10)
	u.broadcaster.SendToUser(providerUserIDStr, payload)
}

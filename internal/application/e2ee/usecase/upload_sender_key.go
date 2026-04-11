package usecase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	e2eePort "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyrequest"
	"go.uber.org/zap"
)

type UploadSenderKeyInput struct {
	RoomID              int64
	ReceiverMemberID    int64
	SenderKeyVersion    int64
	DistributionMessage string // base64
}

type UploadSenderKeyOutput struct{}

type UploadSenderKeyUseCase struct {
	participantRepo      participant.Repository
	chatMemberRepo       chatmember.Repository
	memberSenderKeyRepo  membersenderkey.Repository
	distributionRepo     senderkeydistribution.Repository
	senderKeyRequestRepo senderkeyrequest.Repository
	friendshipRepo       friendship.Repository
	broadcaster          e2eePort.Broadcaster
}

func NewUploadSenderKeyUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	memberSenderKeyRepo membersenderkey.Repository,
	distributionRepo senderkeydistribution.Repository,
	senderKeyRequestRepo senderkeyrequest.Repository,
	friendshipRepo friendship.Repository,
	broadcaster e2eePort.Broadcaster,
) *UploadSenderKeyUseCase {
	return &UploadSenderKeyUseCase{
		participantRepo:      participantRepo,
		chatMemberRepo:       chatMemberRepo,
		memberSenderKeyRepo:  memberSenderKeyRepo,
		distributionRepo:     distributionRepo,
		senderKeyRequestRepo: senderKeyRequestRepo,
		friendshipRepo:       friendshipRepo,
		broadcaster:          broadcaster,
	}
}

func (u *UploadSenderKeyUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[UploadSenderKeyInput],
) (*UploadSenderKeyOutput, error) {
	p, err := u.participantRepo.FindByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, ErrNotRoomMember
	}

	roomID := chatroom.ID(input.Data.RoomID)
	senderMember, err := u.chatMemberRepo.FindByRoomAndParticipant(ctx, roomID, p.ID)
	if err != nil || senderMember.IsDeleted {
		return nil, ErrNotRoomMember
	}

	receiverMember, err := u.chatMemberRepo.FindByID(ctx, chatmember.ID(input.Data.ReceiverMemberID))
	if err != nil || receiverMember.IsDeleted || receiverMember.RoomID != roomID {
		return nil, ErrNotRoomMember
	}

	if blocked, err := u.hasBlockedRelationship(ctx, senderMember.ParticipantID, receiverMember.ParticipantID); err != nil {
		return nil, err
	} else if blocked {
		return nil, ErrForbidden
	}

	distBytes, err := base64.StdEncoding.DecodeString(input.Data.DistributionMessage)
	if err != nil || len(distBytes) == 0 {
		return nil, fmt.Errorf("%w: decode distribution message", ErrInvalidSignature)
	}

	senderMeta := &membersenderkey.MemberSenderKey{
		ChatMemberID:     senderMember.ID,
		SenderKeyVersion: input.Data.SenderKeyVersion,
		ChainID:          membersenderkey.ChainID(input.Data.SenderKeyVersion),
	}
	if err := u.memberSenderKeyRepo.UpsertLatest(ctx, senderMeta); err != nil {
		return nil, fmt.Errorf("upsert sender key metadata: %w", err)
	}

	dist := &senderkeydistribution.SenderKeyDistribution{
		SenderMemberID:      senderMember.ID,
		ReceiverMemberID:    receiverMember.ID,
		RoomID:              int64(roomID),
		SenderKeyVersion:    input.Data.SenderKeyVersion,
		ChainID:             input.Data.SenderKeyVersion,
		DistributionMessage: distBytes,
		Status:              senderkeydistribution.StatusAvailable,
	}
	if err := u.distributionRepo.UpsertAvailable(ctx, dist); err != nil {
		return nil, fmt.Errorf("upsert sender key distribution: %w", err)
	}

	if err := u.senderKeyRequestRepo.MarkFulfilled(ctx, receiverMember.ID, senderMember.ID); err != nil {
		logger.Log.Warn("mark sender key request fulfilled failed",
			zap.Int64("room_id", int64(roomID)),
			zap.Int64("sender_member_id", int64(senderMember.ID)),
			zap.Int64("receiver_member_id", int64(receiverMember.ID)),
			zap.Error(err),
		)
	}
	go u.notifyReceiver(context.Background(), dist, int64(roomID))

	return &UploadSenderKeyOutput{}, nil
}

type wsSenderKeyDistributionAvailablePayload struct {
	RoomID           int64 `json:"room_id"`
	DistributionID   int64 `json:"distribution_id"`
	SenderMemberID   int64 `json:"sender_member_id"`
	ReceiverMemberID int64 `json:"receiver_member_id"`
	SenderKeyVersion int64 `json:"sender_key_version"`
}

func (u *UploadSenderKeyUseCase) notifyReceiver(
	ctx context.Context,
	dist *senderkeydistribution.SenderKeyDistribution,
	roomID int64,
) {
	receiverMember, err := u.chatMemberRepo.FindByID(ctx, dist.ReceiverMemberID)
	if err != nil || receiverMember.IsDeleted {
		return
	}
	receiverParticipant, err := u.participantRepo.FindByID(ctx, receiverMember.ParticipantID)
	if err != nil || receiverParticipant.UserID == nil {
		return
	}
	userIDStr := strconv.FormatInt(int64(*receiverParticipant.UserID), 10)

	payload, err := json.Marshal(struct {
		Type    string                                  `json:"type"`
		Payload wsSenderKeyDistributionAvailablePayload `json:"payload"`
	}{
		Type: "e2ee.sender_key_distribution_available",
		Payload: wsSenderKeyDistributionAvailablePayload{
			RoomID:           roomID,
			DistributionID:   int64(dist.ID),
			SenderMemberID:   int64(dist.SenderMemberID),
			ReceiverMemberID: int64(dist.ReceiverMemberID),
			SenderKeyVersion: dist.SenderKeyVersion,
		},
	})
	if err == nil {
		u.broadcaster.SendToUser(userIDStr, payload)
	}

	// Legacy shim kept during migration.
	legacyPayload, err := json.Marshal(struct {
		Type    string `json:"type"`
		Payload struct {
			RoomID           int64 `json:"room_id"`
			ProviderMemberID int64 `json:"provider_member_id"`
		} `json:"payload"`
	}{
		Type: "e2ee.direct_key_ready",
		Payload: struct {
			RoomID           int64 `json:"room_id"`
			ProviderMemberID int64 `json:"provider_member_id"`
		}{
			RoomID:           roomID,
			ProviderMemberID: int64(dist.SenderMemberID),
		},
	})
	if err == nil {
		u.broadcaster.SendToUser(userIDStr, legacyPayload)
	}
}

func (u *UploadSenderKeyUseCase) hasBlockedRelationship(
	ctx context.Context,
	senderParticipantID, receiverParticipantID participant.ID,
) (bool, error) {
	senderParticipant, err := u.participantRepo.FindByID(ctx, senderParticipantID)
	if err != nil || senderParticipant.UserID == nil {
		return false, nil
	}
	receiverParticipant, err := u.participantRepo.FindByID(ctx, receiverParticipantID)
	if err != nil || receiverParticipant.UserID == nil {
		return false, nil
	}
	rows, err := u.friendshipRepo.FindBetweenUsers(ctx, *senderParticipant.UserID, *receiverParticipant.UserID)
	if err != nil {
		return false, nil
	}
	for _, row := range rows {
		if row.Status == friendship.StatusBlocked {
			return true, nil
		}
	}
	return false, nil
}

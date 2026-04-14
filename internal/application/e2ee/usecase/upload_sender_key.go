package usecase

import (
	"context"
	"encoding/base64"
	"fmt"

	e2eePort "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyrequest"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"go.uber.org/zap"
)

type UploadSenderKeyInput struct {
	RoomID              int64
	ReceiverMemberID    int64
	ReceiverDeviceID    string
	SenderKeyVersion    int64
	DistributionMessage string // base64
}

type UploadSenderKeyOutput struct{}

type UploadSenderKeyUseCase struct {
	participantRepo      participant.Repository
	chatMemberRepo       chatmember.Repository
	accountRepo          account.Repository
	userRepo             user.Repository
	memberSenderKeyRepo  membersenderkey.Repository
	distributionRepo     senderkeydistribution.Repository
	senderKeyRequestRepo senderkeyrequest.Repository
	selfSyncRepo         selfsenderkeysync.Repository
	friendshipRepo       friendship.Repository
	broadcaster          e2eePort.Broadcaster
}

func NewUploadSenderKeyUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	accountRepo account.Repository,
	userRepo user.Repository,
	memberSenderKeyRepo membersenderkey.Repository,
	distributionRepo senderkeydistribution.Repository,
	senderKeyRequestRepo senderkeyrequest.Repository,
	selfSyncRepo selfsenderkeysync.Repository,
	friendshipRepo friendship.Repository,
	broadcaster e2eePort.Broadcaster,
) *UploadSenderKeyUseCase {
	return &UploadSenderKeyUseCase{
		participantRepo:      participantRepo,
		chatMemberRepo:       chatMemberRepo,
		accountRepo:          accountRepo,
		userRepo:             userRepo,
		memberSenderKeyRepo:  memberSenderKeyRepo,
		distributionRepo:     distributionRepo,
		senderKeyRequestRepo: senderKeyRequestRepo,
		selfSyncRepo:         selfSyncRepo,
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
	if err := ensureDeviceNotBlockedByActiveSelfSync(ctx, u.selfSyncRepo, p.ID, input.Base.Request.DeviceID); err != nil {
		return nil, err
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

	senderDeviceID := input.Base.Request.DeviceID

	senderMeta := &membersenderkey.MemberSenderKey{
		ChatMemberID:     senderMember.ID,
		SenderDeviceID:   senderDeviceID,
		SenderKeyVersion: input.Data.SenderKeyVersion,
		ChainID:          membersenderkey.ChainID(input.Data.SenderKeyVersion),
	}
	if err := u.memberSenderKeyRepo.UpsertLatest(ctx, senderMeta); err != nil {
		return nil, fmt.Errorf("upsert sender key metadata: %w", err)
	}

	targetDeviceIDs, err := u.resolveReceiverDeviceIDs(ctx, senderMember, receiverMember, input.Data.ReceiverDeviceID, input.Base.Request.DeviceID)
	if err != nil {
		return nil, err
	}
	if len(targetDeviceIDs) == 0 {
		return &UploadSenderKeyOutput{}, nil
	}

	for _, receiverDeviceID := range targetDeviceIDs {
		if err := ensureDeviceNotBlockedByActiveSelfSync(ctx, u.selfSyncRepo, receiverMember.ParticipantID, receiverDeviceID); err != nil {
			return nil, err
		}
		dist := &senderkeydistribution.SenderKeyDistribution{
			SenderMemberID:      senderMember.ID,
			SenderDeviceID:      senderDeviceID,
			ReceiverMemberID:    receiverMember.ID,
			ReceiverDeviceID:    receiverDeviceID,
			RoomID:              int64(roomID),
			SenderKeyVersion:    input.Data.SenderKeyVersion,
			ChainID:             input.Data.SenderKeyVersion,
			DistributionMessage: distBytes,
			Status:              senderkeydistribution.StatusAvailable,
		}
		if err := u.distributionRepo.UpsertAvailable(ctx, dist); err != nil {
			return nil, fmt.Errorf("upsert sender key distribution: %w", err)
		}

		if err := u.senderKeyRequestRepo.MarkFulfilled(ctx, receiverMember.ID, receiverDeviceID, senderMember.ID, senderDeviceID); err != nil {
			logger.Log.Warn("mark sender key request fulfilled failed",
				zap.Int64("room_id", int64(roomID)),
				zap.Int64("sender_member_id", int64(senderMember.ID)),
				zap.String("sender_device_id", senderDeviceID.String()),
				zap.Int64("receiver_member_id", int64(receiverMember.ID)),
				zap.String("receiver_device_id", receiverDeviceID.String()),
				zap.Error(err),
			)
		}
		go notifySenderKeyDistributionAvailable(
			context.Background(),
			u.broadcaster,
			u.participantRepo,
			u.chatMemberRepo,
			dist,
			int64(roomID),
		)
	}

	return &UploadSenderKeyOutput{}, nil
}

func (u *UploadSenderKeyUseCase) resolveReceiverDeviceIDs(
	ctx context.Context,
	senderMember, receiverMember *chatmember.ChatMember,
	explicitReceiverDeviceID string,
	currentDeviceID shared.DeviceID,
) ([]shared.DeviceID, error) {
	if explicitReceiverDeviceID != "" {
		deviceID, err := shared.ParseDeviceID(explicitReceiverDeviceID)
		if err != nil {
			return nil, fmt.Errorf("%w: receiver device id", ErrInvalidSignature)
		}
		return []shared.DeviceID{deviceID}, nil
	}

	receiverParticipant, err := u.participantRepo.FindByID(ctx, receiverMember.ParticipantID)
	if err != nil || receiverParticipant.UserID == nil {
		return nil, nil
	}
	receiverUser, err := u.userRepo.FindByID(ctx, *receiverParticipant.UserID)
	if err != nil {
		return nil, fmt.Errorf("find receiver user: %w", err)
	}
	receiverAccount, err := u.accountRepo.FindByID(ctx, receiverUser.AccountID)
	if err != nil {
		return nil, fmt.Errorf("find receiver account: %w", err)
	}

	targets := make([]shared.DeviceID, 0, len(receiverAccount.Devices))
	for _, deviceBinding := range receiverAccount.Devices {
		if deviceBinding.Status != account.DeviceStatusReady {
			continue
		}
		if receiverMember.ParticipantID == senderMember.ParticipantID && deviceBinding.DeviceID == currentDeviceID {
			continue
		}
		targets = append(targets, deviceBinding.DeviceID)
	}
	return targets, nil
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

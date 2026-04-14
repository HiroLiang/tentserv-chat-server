package usecase

import (
	"context"
	"errors"
	"fmt"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyreceipt"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type GetSenderKeyDistributionStatusInput struct {
	RoomID int64
}

type SenderKeyDeviceRef struct {
	MemberID int64
	DeviceID string
}

type GetSenderKeyDistributionStatusOutput struct {
	OwnDeviceSenderKeyExists bool
	RequestableSources       []SenderKeyDeviceRef
	AvailableFromSources     []SenderKeyDeviceRef
	AvailableToTargets       []SenderKeyDeviceRef
	PendingReceivers         []SenderKeyDeviceRef
	PendingFromSources       []SenderKeyDeviceRef
}

type GetSenderKeyDistributionStatusUseCase struct {
	participantRepo     participant.Repository
	chatMemberRepo      chatmember.Repository
	accountRepo         account.Repository
	userRepo            user.Repository
	memberSenderKeyRepo membersenderkey.Repository
	distributionRepo    senderkeydistribution.Repository
	receiptRepo         senderkeyreceipt.Repository
}

func NewGetSenderKeyDistributionStatusUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	accountRepo account.Repository,
	userRepo user.Repository,
	memberSenderKeyRepo membersenderkey.Repository,
	distributionRepo senderkeydistribution.Repository,
	receiptRepo senderkeyreceipt.Repository,
) *GetSenderKeyDistributionStatusUseCase {
	return &GetSenderKeyDistributionStatusUseCase{
		participantRepo:     participantRepo,
		chatMemberRepo:      chatMemberRepo,
		accountRepo:         accountRepo,
		userRepo:            userRepo,
		memberSenderKeyRepo: memberSenderKeyRepo,
		distributionRepo:    distributionRepo,
		receiptRepo:         receiptRepo,
	}
}

func (u *GetSenderKeyDistributionStatusUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[GetSenderKeyDistributionStatusInput],
) (*GetSenderKeyDistributionStatusOutput, error) {
	callerParticipant, err := u.participantRepo.FindByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, ErrNotRoomMember
	}

	roomID := chatroom.ID(input.Data.RoomID)
	callerMember, err := u.chatMemberRepo.FindByRoomAndParticipant(ctx, roomID, callerParticipant.ID)
	if err != nil || callerMember.IsDeleted {
		return nil, ErrNotRoomMember
	}

	members, err := u.chatMemberRepo.FindByRoom(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("find room members: %w", err)
	}

	out := &GetSenderKeyDistributionStatusOutput{
		RequestableSources:   []SenderKeyDeviceRef{},
		AvailableFromSources: []SenderKeyDeviceRef{},
		AvailableToTargets:   []SenderKeyDeviceRef{},
		PendingReceivers:     []SenderKeyDeviceRef{},
		PendingFromSources:   []SenderKeyDeviceRef{},
	}

	currentDeviceID := input.Base.Request.DeviceID
	ownLatestKeys, err := u.memberSenderKeyRepo.FindLatestForMember(ctx, callerMember.ID)
	if err != nil && !errors.Is(err, membersenderkey.ErrNotFound) {
		return nil, fmt.Errorf("find own sender keys: %w", err)
	}

	var ownCurrentDeviceKey *membersenderkey.MemberSenderKey
	for _, key := range ownLatestKeys {
		if key.SenderDeviceID == currentDeviceID {
			ownCurrentDeviceKey = key
			out.OwnDeviceSenderKeyExists = true
			break
		}
	}

	for _, member := range members {
		if member.IsDeleted || member.ID == callerMember.ID {
			continue
		}

		peerKeys, err := u.memberSenderKeyRepo.FindLatestForMember(ctx, member.ID)
		if err != nil && !errors.Is(err, membersenderkey.ErrNotFound) {
			return nil, fmt.Errorf("find peer sender keys for member %d: %w", member.ID, err)
		}

		for _, peerKey := range peerKeys {
			ref := SenderKeyDeviceRef{MemberID: int64(member.ID), DeviceID: peerKey.SenderDeviceID.String()}
			if receipt, receiptErr := u.receiptRepo.FindLatest(ctx, member.ID, peerKey.SenderDeviceID, callerMember.ID, currentDeviceID); receiptErr == nil && receipt.SenderKeyVersion >= peerKey.SenderKeyVersion {
				continue
			} else if receiptErr != nil && !errors.Is(receiptErr, senderkeyreceipt.ErrNotFound) {
				return nil, fmt.Errorf("find sender key receipt from member %d device %s: %w", member.ID, peerKey.SenderDeviceID.String(), receiptErr)
			}
			dist, distErr := u.distributionRepo.FindLatest(ctx, member.ID, peerKey.SenderDeviceID, callerMember.ID, currentDeviceID)
			switch {
			case errors.Is(distErr, senderkeydistribution.ErrNotFound):
				out.RequestableSources = append(out.RequestableSources, ref)
				out.PendingFromSources = append(out.PendingFromSources, ref)
			case distErr != nil:
				return nil, fmt.Errorf("find latest distribution from member %d device %s: %w", member.ID, peerKey.SenderDeviceID.String(), distErr)
			case dist.SenderKeyVersion < peerKey.SenderKeyVersion || dist.Status == senderkeydistribution.StatusFailed:
				out.RequestableSources = append(out.RequestableSources, ref)
				out.PendingFromSources = append(out.PendingFromSources, ref)
			case dist.Status == senderkeydistribution.StatusAvailable:
				out.AvailableFromSources = append(out.AvailableFromSources, ref)
			}
		}

		if ownCurrentDeviceKey == nil {
			continue
		}

		targetDeviceIDs, err := u.resolveReadyDeviceIDs(ctx, member.ParticipantID)
		if err != nil {
			return nil, err
		}
		for _, receiverDeviceID := range targetDeviceIDs {
			ref := SenderKeyDeviceRef{MemberID: int64(member.ID), DeviceID: receiverDeviceID.String()}
			if receipt, receiptErr := u.receiptRepo.FindLatest(ctx, callerMember.ID, currentDeviceID, member.ID, receiverDeviceID); receiptErr == nil && receipt.SenderKeyVersion >= ownCurrentDeviceKey.SenderKeyVersion {
				continue
			} else if receiptErr != nil && !errors.Is(receiptErr, senderkeyreceipt.ErrNotFound) {
				return nil, fmt.Errorf("find sender key receipt from caller %d device %s to member %d device %s: %w", callerMember.ID, currentDeviceID.String(), member.ID, receiverDeviceID.String(), receiptErr)
			}
			dist, distErr := u.distributionRepo.FindLatest(ctx, callerMember.ID, currentDeviceID, member.ID, receiverDeviceID)
			switch {
			case errors.Is(distErr, senderkeydistribution.ErrNotFound):
				out.PendingReceivers = append(out.PendingReceivers, ref)
			case distErr != nil:
				return nil, fmt.Errorf("find latest distribution from caller %d device %s to member %d device %s: %w", callerMember.ID, currentDeviceID.String(), member.ID, receiverDeviceID.String(), distErr)
			case dist.SenderKeyVersion < ownCurrentDeviceKey.SenderKeyVersion || dist.Status == senderkeydistribution.StatusFailed:
				out.PendingReceivers = append(out.PendingReceivers, ref)
			case dist.Status == senderkeydistribution.StatusAvailable:
				out.AvailableToTargets = append(out.AvailableToTargets, ref)
			}
		}
	}

	return out, nil
}

func (u *GetSenderKeyDistributionStatusUseCase) resolveReadyDeviceIDs(
	ctx context.Context,
	participantID participant.ID,
) ([]shared.DeviceID, error) {
	p, err := u.participantRepo.FindByID(ctx, participantID)
	if err != nil || p.UserID == nil {
		return nil, nil
	}
	userData, err := u.userRepo.FindByID(ctx, *p.UserID)
	if err != nil {
		return nil, fmt.Errorf("find participant user: %w", err)
	}
	accountData, err := u.accountRepo.FindByID(ctx, userData.AccountID)
	if err != nil {
		return nil, fmt.Errorf("find participant account: %w", err)
	}

	deviceIDs := make([]shared.DeviceID, 0, len(accountData.Devices))
	for _, deviceBinding := range accountData.Devices {
		if deviceBinding.Status == account.DeviceStatusReady {
			deviceIDs = append(deviceIDs, deviceBinding.DeviceID)
		}
	}
	return deviceIDs, nil
}

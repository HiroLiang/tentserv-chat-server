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

type SenderKeyRouteRef struct {
	UserID   int64
	MemberID int64
	DeviceID string
}

type GetSenderKeyDistributionStatusOutput struct {
	OwnMemberSenderKeyExists bool
	RequestableSources       []SenderKeyRouteRef
	AvailableFromSources     []SenderKeyRouteRef
	AvailableToTargets       []SenderKeyRouteRef
	PendingReceivers         []SenderKeyRouteRef
	PendingFromSources       []SenderKeyRouteRef
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
		RequestableSources:   []SenderKeyRouteRef{},
		AvailableFromSources: []SenderKeyRouteRef{},
		AvailableToTargets:   []SenderKeyRouteRef{},
		PendingReceivers:     []SenderKeyRouteRef{},
		PendingFromSources:   []SenderKeyRouteRef{},
	}

	currentDeviceID := input.Base.Request.DeviceID
	ownLatestKey, err := u.memberSenderKeyRepo.FindLatest(ctx, callerMember.ID)
	if err != nil && !errors.Is(err, membersenderkey.ErrNotFound) {
		return nil, fmt.Errorf("find own sender keys: %w", err)
	}
	if err == nil {
		out.OwnMemberSenderKeyExists = true
	}

	for _, member := range members {
		if member.IsDeleted || member.ID == callerMember.ID {
			continue
		}

		routeState, err := u.resolveRouteState(ctx, member)
		if err != nil {
			return nil, err
		}
		if routeState == nil {
			continue
		}

		peerKey, err := u.memberSenderKeyRepo.FindLatest(ctx, member.ID)
		if err != nil && !errors.Is(err, membersenderkey.ErrNotFound) {
			return nil, fmt.Errorf("find peer sender keys for member %d: %w", member.ID, err)
		}
		if err == nil {
			receipt, receiptErr := u.receiptRepo.FindLatest(ctx, member.ID, currentDeviceID)
			switch {
			case receiptErr == nil && receipt.SenderKeyVersion >= peerKey.SenderKeyVersion:
			case receiptErr != nil && !errors.Is(receiptErr, senderkeyreceipt.ErrNotFound):
				return nil, fmt.Errorf("find sender key receipt from member %d to device %s: %w", member.ID, currentDeviceID.String(), receiptErr)
			default:
				dist, distErr := u.distributionRepo.FindLatestForReceiver(ctx, member.ID, callerMember.ID, currentDeviceID)
				switch {
				case distErr == nil && dist.SenderKeyVersion >= peerKey.SenderKeyVersion && dist.Status == senderkeydistribution.StatusAvailable:
					out.AvailableFromSources = append(out.AvailableFromSources, SenderKeyRouteRef{
						UserID:   routeState.userID,
						MemberID: int64(member.ID),
						DeviceID: dist.SenderDeviceID.String(),
					})
				case distErr == nil && dist.SenderKeyVersion >= peerKey.SenderKeyVersion && dist.Status == senderkeydistribution.StatusConsumed:
				case distErr != nil && !errors.Is(distErr, senderkeydistribution.ErrNotFound):
					return nil, fmt.Errorf("find latest distribution from member %d to device %s: %w", member.ID, currentDeviceID.String(), distErr)
				default:
					for _, providerDeviceID := range routeState.readyDeviceIDs {
						ref := SenderKeyRouteRef{
							UserID:   routeState.userID,
							MemberID: int64(member.ID),
							DeviceID: providerDeviceID.String(),
						}
						out.RequestableSources = append(out.RequestableSources, ref)
						out.PendingFromSources = append(out.PendingFromSources, ref)
					}
				}
			}
		}

		if ownLatestKey == nil {
			continue
		}

		for _, receiverDeviceID := range routeState.readyDeviceIDs {
			ref := SenderKeyRouteRef{
				UserID:   routeState.userID,
				MemberID: int64(member.ID),
				DeviceID: receiverDeviceID.String(),
			}
			if receipt, receiptErr := u.receiptRepo.FindLatest(ctx, callerMember.ID, receiverDeviceID); receiptErr == nil && receipt.SenderKeyVersion >= ownLatestKey.SenderKeyVersion {
				continue
			} else if receiptErr != nil && !errors.Is(receiptErr, senderkeyreceipt.ErrNotFound) {
				return nil, fmt.Errorf("find sender key receipt from caller %d to member %d device %s: %w", callerMember.ID, member.ID, receiverDeviceID.String(), receiptErr)
			}
			dist, distErr := u.distributionRepo.FindLatestForReceiver(ctx, callerMember.ID, member.ID, receiverDeviceID)
			switch {
			case errors.Is(distErr, senderkeydistribution.ErrNotFound):
				out.PendingReceivers = append(out.PendingReceivers, ref)
			case distErr != nil:
				return nil, fmt.Errorf("find latest distribution from caller %d to member %d device %s: %w", callerMember.ID, member.ID, receiverDeviceID.String(), distErr)
			case dist.SenderKeyVersion < ownLatestKey.SenderKeyVersion || dist.Status == senderkeydistribution.StatusFailed:
				out.PendingReceivers = append(out.PendingReceivers, ref)
			case dist.Status == senderkeydistribution.StatusAvailable:
				out.AvailableToTargets = append(out.AvailableToTargets, ref)
			}
		}
	}

	return out, nil
}

type senderKeyRouteState struct {
	userID         int64
	readyDeviceIDs []shared.DeviceID
}

func (u *GetSenderKeyDistributionStatusUseCase) resolveRouteState(
	ctx context.Context,
	member *chatmember.ChatMember,
) (*senderKeyRouteState, error) {
	p, err := u.participantRepo.FindByID(ctx, member.ParticipantID)
	if err != nil || p.UserID == nil {
		return nil, nil
	}

	deviceIDs, err := u.resolveReadyDeviceIDs(ctx, member.ParticipantID)
	if err != nil {
		return nil, err
	}

	return &senderKeyRouteState{
		userID:         int64(*p.UserID),
		readyDeviceIDs: deviceIDs,
	}, nil
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

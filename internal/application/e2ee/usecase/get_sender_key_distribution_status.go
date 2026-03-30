package usecase

import (
	"context"
	"fmt"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
)

type GetSenderKeyDistributionStatusInput struct {
	RoomID int64
}

type GetSenderKeyDistributionStatusOutput struct {
	// PendingReceivers: member IDs who have not yet fetched my latest sender key.
	PendingReceivers []int64
	// PendingFromMembers: member IDs whose latest sender key I have not yet fetched.
	PendingFromMembers []int64
}

type GetSenderKeyDistributionStatusUseCase struct {
	participantRepo     participant.Repository
	chatMemberRepo      chatmember.Repository
	memberSenderKeyRepo membersenderkey.Repository
	distributionRepo    senderkeydistribution.Repository
}

func NewGetSenderKeyDistributionStatusUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	memberSenderKeyRepo membersenderkey.Repository,
	distributionRepo senderkeydistribution.Repository,
) *GetSenderKeyDistributionStatusUseCase {
	return &GetSenderKeyDistributionStatusUseCase{
		participantRepo:     participantRepo,
		chatMemberRepo:      chatMemberRepo,
		memberSenderKeyRepo: memberSenderKeyRepo,
		distributionRepo:    distributionRepo,
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

	callerMember, err := u.chatMemberRepo.FindByRoomAndParticipant(ctx, chatroom.ID(input.Data.RoomID), callerParticipant.ID)
	if err != nil {
		return nil, ErrNotRoomMember
	}

	// Pending receivers: who hasn't fetched my latest key yet?
	pendingReceivers, err := u.pendingReceivers(ctx, callerMember.ID)
	if err != nil {
		return nil, err
	}

	// Pending from members: whose key haven't I fetched yet?
	pendingFrom, err := u.pendingFromMembers(ctx, chatroom.ID(input.Data.RoomID), callerMember.ID)
	if err != nil {
		return nil, err
	}

	return &GetSenderKeyDistributionStatusOutput{
		PendingReceivers:   pendingReceivers,
		PendingFromMembers: pendingFrom,
	}, nil
}

func (u *GetSenderKeyDistributionStatusUseCase) pendingReceivers(
	ctx context.Context,
	callerMemberID chatmember.ID,
) ([]int64, error) {
	latest, err := u.memberSenderKeyRepo.FindLatest(ctx, callerMemberID)
	if err != nil {
		if err == membersenderkey.ErrNotFound {
			return []int64{}, nil
		}
		return nil, fmt.Errorf("find latest sender key: %w", err)
	}

	pending, err := u.distributionRepo.FindPendingReceivers(ctx, callerMemberID, int(latest.ChainID))
	if err != nil {
		return nil, fmt.Errorf("find pending receivers: %w", err)
	}

	ids := make([]int64, len(pending))
	for i, id := range pending {
		ids[i] = int64(id)
	}
	return ids, nil
}

func (u *GetSenderKeyDistributionStatusUseCase) pendingFromMembers(
	ctx context.Context,
	roomID chatroom.ID,
	callerMemberID chatmember.ID,
) ([]int64, error) {
	members, err := u.chatMemberRepo.FindByRoom(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("find room members: %w", err)
	}

	var pending []int64
	for _, m := range members {
		if m.ID == callerMemberID || m.IsDeleted {
			continue
		}
		latest, err := u.memberSenderKeyRepo.FindLatest(ctx, m.ID)
		if err != nil {
			if err == membersenderkey.ErrNotFound {
				continue
			}
			return nil, fmt.Errorf("find latest sender key for member %d: %w", m.ID, err)
		}

		senderPending, err := u.distributionRepo.FindPendingReceivers(ctx, m.ID, int(latest.ChainID))
		if err != nil {
			return nil, err
		}
		for _, receiverID := range senderPending {
			if receiverID == callerMemberID {
				pending = append(pending, int64(m.ID))
				break
			}
		}
	}
	return pending, nil
}

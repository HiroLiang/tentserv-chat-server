package usecase

import (
	"context"
	"errors"
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
	OwnSenderKeyExists     bool
	RequestableMemberIDs   []int64
	AvailableFromMemberIDs []int64
	PendingReceivers       []int64
	// Legacy compatibility for still-migrating callers.
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
		RequestableMemberIDs:   []int64{},
		AvailableFromMemberIDs: []int64{},
		PendingReceivers:       []int64{},
		PendingFromMembers:     []int64{},
	}

	ownLatest, err := u.memberSenderKeyRepo.FindLatest(ctx, callerMember.ID)
	if err == nil {
		out.OwnSenderKeyExists = true
	}

	for _, member := range members {
		if member.IsDeleted || member.ID == callerMember.ID {
			continue
		}

		latest, latestErr := u.memberSenderKeyRepo.FindLatest(ctx, member.ID)
		switch {
		case latestErr == nil:
			dist, distErr := u.distributionRepo.FindLatest(ctx, member.ID, callerMember.ID)
			if errors.Is(distErr, senderkeydistribution.ErrNotFound) {
				out.RequestableMemberIDs = append(out.RequestableMemberIDs, int64(member.ID))
				out.PendingFromMembers = append(out.PendingFromMembers, int64(member.ID))
			} else if distErr != nil {
				return nil, fmt.Errorf("find latest distribution from member %d to caller %d: %w", member.ID, callerMember.ID, distErr)
			} else if dist.SenderKeyVersion < latest.SenderKeyVersion || dist.Status == senderkeydistribution.StatusFailed {
				out.RequestableMemberIDs = append(out.RequestableMemberIDs, int64(member.ID))
				out.PendingFromMembers = append(out.PendingFromMembers, int64(member.ID))
			} else if dist.Status == senderkeydistribution.StatusAvailable {
				out.AvailableFromMemberIDs = append(out.AvailableFromMemberIDs, int64(member.ID))
			}
		case errors.Is(latestErr, membersenderkey.ErrNotFound):
			out.RequestableMemberIDs = append(out.RequestableMemberIDs, int64(member.ID))
			out.PendingFromMembers = append(out.PendingFromMembers, int64(member.ID))
		default:
			return nil, fmt.Errorf("find latest sender key for member %d: %w", member.ID, latestErr)
		}

		if out.OwnSenderKeyExists {
			dist, distErr := u.distributionRepo.FindLatest(ctx, callerMember.ID, member.ID)
			if errors.Is(distErr, senderkeydistribution.ErrNotFound) {
				out.PendingReceivers = append(out.PendingReceivers, int64(member.ID))
			} else if distErr != nil {
				return nil, fmt.Errorf("find latest distribution from caller %d to member %d: %w", callerMember.ID, member.ID, distErr)
			} else if dist.SenderKeyVersion < ownLatest.SenderKeyVersion || dist.Status != senderkeydistribution.StatusConsumed {
				out.PendingReceivers = append(out.PendingReceivers, int64(member.ID))
			}
		}
	}

	return out, nil
}

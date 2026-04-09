package usecase

import (
	"context"
	"encoding/base64"
	"fmt"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
)

type GetPendingSenderKeyDistributionsInput struct {
	RoomID int64
}

type PendingSenderKeyDistributionItem struct {
	DistributionID      int64  `json:"distribution_id"`
	SenderMemberID      int64  `json:"sender_member_id"`
	ReceiverMemberID    int64  `json:"receiver_member_id"`
	SenderKeyVersion    int64  `json:"sender_key_version"`
	DistributionMessage string `json:"distribution_message"`
}

type GetPendingSenderKeyDistributionsOutput struct {
	Distributions []PendingSenderKeyDistributionItem `json:"distributions"`
}

type GetPendingSenderKeyDistributionsUseCase struct {
	participantRepo  participant.Repository
	chatMemberRepo   chatmember.Repository
	distributionRepo senderkeydistribution.Repository
}

func NewGetPendingSenderKeyDistributionsUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	distributionRepo senderkeydistribution.Repository,
) *GetPendingSenderKeyDistributionsUseCase {
	return &GetPendingSenderKeyDistributionsUseCase{
		participantRepo:  participantRepo,
		chatMemberRepo:   chatMemberRepo,
		distributionRepo: distributionRepo,
	}
}

func (u *GetPendingSenderKeyDistributionsUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[GetPendingSenderKeyDistributionsInput],
) (*GetPendingSenderKeyDistributionsOutput, error) {
	callerParticipant, err := u.participantRepo.FindByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, ErrNotRoomMember
	}

	roomID := chatroom.ID(input.Data.RoomID)
	callerMember, err := u.chatMemberRepo.FindByRoomAndParticipant(ctx, roomID, callerParticipant.ID)
	if err != nil || callerMember.IsDeleted {
		return nil, ErrNotRoomMember
	}

	dists, err := u.distributionRepo.FindAvailableByRoomAndReceiver(ctx, roomID, callerMember.ID)
	if err != nil {
		return nil, fmt.Errorf("find pending sender key distributions: %w", err)
	}

	items := make([]PendingSenderKeyDistributionItem, 0, len(dists))
	for _, dist := range dists {
		items = append(items, PendingSenderKeyDistributionItem{
			DistributionID:      int64(dist.ID),
			SenderMemberID:      int64(dist.SenderMemberID),
			ReceiverMemberID:    int64(dist.ReceiverMemberID),
			SenderKeyVersion:    dist.SenderKeyVersion,
			DistributionMessage: base64.StdEncoding.EncodeToString(dist.DistributionMessage),
		})
	}

	return &GetPendingSenderKeyDistributionsOutput{Distributions: items}, nil
}

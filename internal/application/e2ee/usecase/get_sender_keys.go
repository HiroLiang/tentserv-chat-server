package usecase

import (
	"context"
	"encoding/base64"
	"fmt"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
)

type GetSenderKeysInput struct {
	RoomID int64
}

type SenderKeyItem struct {
	ChatMemberID        int64
	SenderKeyPublic     string // base64
	DistributionMessage string // base64
}

type GetSenderKeysOutput struct {
	Keys []SenderKeyItem
}

type GetSenderKeysUseCase struct {
	participantRepo     participant.Repository
	chatMemberRepo      chatmember.Repository
	memberSenderKeyRepo membersenderkey.Repository
	distributionRepo    senderkeydistribution.Repository
}

func NewGetSenderKeysUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	memberSenderKeyRepo membersenderkey.Repository,
	distributionRepo senderkeydistribution.Repository,
) *GetSenderKeysUseCase {
	return &GetSenderKeysUseCase{
		participantRepo:     participantRepo,
		chatMemberRepo:      chatMemberRepo,
		memberSenderKeyRepo: memberSenderKeyRepo,
		distributionRepo:    distributionRepo,
	}
}

func (u *GetSenderKeysUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[GetSenderKeysInput],
) (*GetSenderKeysOutput, error) {
	// Verify the caller is a member of the room
	callerParticipant, err := u.participantRepo.FindByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		return nil, ErrNotRoomMember
	}

	callerMember, err := u.chatMemberRepo.FindByRoomAndParticipant(ctx, chatroom.ID(input.Data.RoomID), callerParticipant.ID)
	if err != nil {
		return nil, ErrNotRoomMember
	}

	members, err := u.chatMemberRepo.FindByRoom(ctx, chatroom.ID(input.Data.RoomID))
	if err != nil {
		return nil, fmt.Errorf("find room members: %w", err)
	}

	memberIDs := make([]chatmember.ID, 0, len(members))
	for _, m := range members {
		memberIDs = append(memberIDs, m.ID)
	}

	senderKeys, err := u.memberSenderKeyRepo.FindAllByMembers(ctx, memberIDs)
	if err != nil {
		return nil, fmt.Errorf("find sender keys: %w", err)
	}

	items := make([]SenderKeyItem, 0, len(senderKeys))
	for _, sk := range senderKeys {
		items = append(items, SenderKeyItem{
			ChatMemberID:        int64(sk.ChatMemberID),
			SenderKeyPublic:     base64.StdEncoding.EncodeToString(sk.SenderKeyPublic[:]),
			DistributionMessage: base64.StdEncoding.EncodeToString(sk.DistributionMessage),
		})
	}

	// Record ACK: caller has fetched each sender's key at its current chain_id.
	// Fire best-effort; do not fail the response if this write fails.
	go u.recordDistributions(context.Background(), callerMember.ID, senderKeys)

	return &GetSenderKeysOutput{Keys: items}, nil
}

func (u *GetSenderKeysUseCase) recordDistributions(
	ctx context.Context,
	receiverMemberID chatmember.ID,
	keys []*membersenderkey.MemberSenderKey,
) {
	if len(keys) == 0 {
		return
	}

	dists := make([]*senderkeydistribution.SenderKeyDistribution, 0, len(keys))
	for _, sk := range keys {
		if sk.ChatMemberID == receiverMemberID {
			continue // do not record self-fetching own key
		}
		dists = append(dists, &senderkeydistribution.SenderKeyDistribution{
			SenderMemberID:   sk.ChatMemberID,
			ReceiverMemberID: receiverMemberID,
			ChainID:          int(sk.ChainID),
		})
	}

	_ = u.distributionRepo.UpsertBatch(ctx, dists)
}

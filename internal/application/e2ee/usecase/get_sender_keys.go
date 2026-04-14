package usecase

import (
	"context"
	"fmt"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyreceipt"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type GetSenderKeysInput struct {
	RoomID int64
}

type SenderKeyItem struct {
	ChatMemberID     int64
	ProviderDeviceID string
	SenderKeyVersion int64
}

type GetSenderKeysOutput struct {
	Keys []SenderKeyItem
}

type GetSenderKeysUseCase struct {
	participantRepo     participant.Repository
	chatMemberRepo      chatmember.Repository
	memberSenderKeyRepo membersenderkey.Repository
	receiptRepo         senderkeyreceipt.Repository
}

func NewGetSenderKeysUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	memberSenderKeyRepo membersenderkey.Repository,
	receiptRepo senderkeyreceipt.Repository,
) *GetSenderKeysUseCase {
	return &GetSenderKeysUseCase{
		participantRepo:     participantRepo,
		chatMemberRepo:      chatMemberRepo,
		memberSenderKeyRepo: memberSenderKeyRepo,
		receiptRepo:         receiptRepo,
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
	if err != nil || callerMember.IsDeleted {
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
			ChatMemberID:     int64(sk.ChatMemberID),
			ProviderDeviceID: sk.SenderDeviceID.String(),
			SenderKeyVersion: sk.SenderKeyVersion,
		})
	}

	// Record ACK: caller has fetched each sender's key at its current sender-key version.
	// Fire best-effort; do not fail the response if this write fails.
	go u.recordReceipts(context.Background(), callerMember.ID, input.Base.Request.DeviceID, senderKeys)

	return &GetSenderKeysOutput{Keys: items}, nil
}

func (u *GetSenderKeysUseCase) recordReceipts(
	ctx context.Context,
	receiverMemberID chatmember.ID,
	receiverDeviceID shared.DeviceID,
	keys []*membersenderkey.MemberSenderKey,
) {
	if len(keys) == 0 {
		return
	}

	for _, sk := range keys {
		if sk.ChatMemberID == receiverMemberID {
			continue // do not record self-fetching own key
		}
		_ = u.receiptRepo.Upsert(ctx, &senderkeyreceipt.SenderKeyReceipt{
			SenderMemberID:   sk.ChatMemberID,
			SenderDeviceID:   sk.SenderDeviceID,
			ReceiverMemberID: receiverMemberID,
			ReceiverDeviceID: receiverDeviceID,
			SenderKeyVersion: sk.SenderKeyVersion,
			Source:           senderkeyreceipt.SourceSenderKeys,
		})
	}
}

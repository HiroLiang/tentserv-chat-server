package usecase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	e2eePort "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyrequest"
)

type UploadSenderKeyInput struct {
	RoomID              int64
	SenderKeyPublic     string // base64
	DistributionMessage string // base64
}

type UploadSenderKeyOutput struct{}

type UploadSenderKeyUseCase struct {
	participantRepo      participant.Repository
	chatMemberRepo       chatmember.Repository
	memberSenderKeyRepo  membersenderkey.Repository
	senderKeyRequestRepo senderkeyrequest.Repository
	broadcaster          e2eePort.Broadcaster
}

func NewUploadSenderKeyUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	memberSenderKeyRepo membersenderkey.Repository,
	senderKeyRequestRepo senderkeyrequest.Repository,
	broadcaster e2eePort.Broadcaster,
) *UploadSenderKeyUseCase {
	return &UploadSenderKeyUseCase{
		participantRepo:      participantRepo,
		chatMemberRepo:       chatMemberRepo,
		memberSenderKeyRepo:  memberSenderKeyRepo,
		senderKeyRequestRepo: senderKeyRequestRepo,
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

	member, err := u.chatMemberRepo.FindByRoomAndParticipant(ctx, chatroom.ID(input.Data.RoomID), p.ID)
	if err != nil {
		return nil, ErrNotRoomMember
	}

	pubBytes, err := base64.StdEncoding.DecodeString(input.Data.SenderKeyPublic)
	if err != nil || len(pubBytes) != 32 {
		return nil, fmt.Errorf("%w: decode sender key public", ErrInvalidSignature)
	}

	distBytes, err := base64.StdEncoding.DecodeString(input.Data.DistributionMessage)
	if err != nil {
		return nil, fmt.Errorf("%w: decode distribution message", ErrInvalidSignature)
	}

	var pub membersenderkey.SenderKeyPublic
	copy(pub[:], pubBytes)

	sk := &membersenderkey.MemberSenderKey{
		ChatMemberID:        member.ID,
		SenderKeyPublic:     pub,
		DistributionMessage: distBytes,
	}

	if err := u.memberSenderKeyRepo.Add(ctx, sk); err != nil {
		return nil, fmt.Errorf("add sender key: %w", err)
	}

	// Notify pending requesters and fulfill their requests in the background.
	go u.notifyRequesters(context.Background(), member.ID, input.Data.RoomID)

	return &UploadSenderKeyOutput{}, nil
}

type wsDirectKeyReadyPayload struct {
	RoomID           int64 `json:"room_id"`
	ProviderMemberID int64 `json:"provider_member_id"`
}

// notifyRequesters looks up pending sender_key_requests for this provider,
// sends e2ee.direct_key_ready to each requester, and marks those requests fulfilled.
func (u *UploadSenderKeyUseCase) notifyRequesters(ctx context.Context, providerMemberID chatmember.ID, roomID int64) {
	requests, err := u.senderKeyRequestRepo.FindPendingByProvider(ctx, providerMemberID)
	if err != nil || len(requests) == 0 {
		return
	}

	payload, err := json.Marshal(struct {
		Type    string                  `json:"type"`
		Payload wsDirectKeyReadyPayload `json:"payload"`
	}{
		Type: "e2ee.direct_key_ready",
		Payload: wsDirectKeyReadyPayload{
			RoomID:           roomID,
			ProviderMemberID: int64(providerMemberID),
		},
	})
	if err != nil {
		return
	}

	for _, req := range requests {
		// Find the requester's participant to get their user ID for WS routing.
		requesterMember, err := u.chatMemberRepo.FindByID(ctx, req.RequesterMemberID)
		if err != nil {
			continue
		}
		requesterParticipant, err := u.participantRepo.FindByID(ctx, requesterMember.ParticipantID)
		if err != nil || requesterParticipant.UserID == nil {
			continue
		}
		userIDStr := strconv.FormatInt(int64(*requesterParticipant.UserID), 10)
		u.broadcaster.SendToUser(userIDStr, payload)

		_ = u.senderKeyRequestRepo.MarkFulfilled(ctx, req.RequesterMemberID, providerMemberID)
	}
}

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
)

type UploadSenderKeyInput struct {
	RoomID              int64
	SenderKeyPublic     string // base64
	DistributionMessage string // base64
}

type UploadSenderKeyOutput struct{}

type UploadSenderKeyUseCase struct {
	participantRepo     participant.Repository
	chatMemberRepo      chatmember.Repository
	memberSenderKeyRepo membersenderkey.Repository
}

func NewUploadSenderKeyUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	memberSenderKeyRepo membersenderkey.Repository,
) *UploadSenderKeyUseCase {
	return &UploadSenderKeyUseCase{
		participantRepo:     participantRepo,
		chatMemberRepo:      chatMemberRepo,
		memberSenderKeyRepo: memberSenderKeyRepo,
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

	return &UploadSenderKeyOutput{}, nil
}

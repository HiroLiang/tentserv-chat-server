package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
)

type UpdateMemberStatusInput struct {
	RoomID int64
}

type MemberStatusInfo struct {
	MemberID   int64
	LastReadAt *time.Time
}

type UpdateMemberStatusOutput struct {
	Members []MemberStatusInfo
}

type UpdateMemberStatusUseCase struct {
	participantRepo participant.Repository
	chatMemberRepo  chatmember.Repository
}

func NewUpdateMemberStatusUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
) *UpdateMemberStatusUseCase {
	return &UpdateMemberStatusUseCase{
		participantRepo: participantRepo,
		chatMemberRepo:  chatMemberRepo,
	}
}

func (uc *UpdateMemberStatusUseCase) Execute(
	ctx context.Context,
	input shared.UseCaseInput[UpdateMemberStatusInput],
) (UpdateMemberStatusOutput, error) {
	callerParticipant, err := uc.participantRepo.FindByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return UpdateMemberStatusOutput{}, ErrParticipantNotFound
		}
		return UpdateMemberStatusOutput{}, err
	}

	roomID := chatroom.ID(input.Data.RoomID)

	callerMember, err := uc.chatMemberRepo.FindByRoomAndParticipant(ctx, roomID, callerParticipant.ID)
	if err != nil || callerMember.IsDeleted {
		return UpdateMemberStatusOutput{}, ErrNotRoomMember
	}

	now := time.Now()
	callerMember.LastReadAt = &now
	if err := uc.chatMemberRepo.Update(ctx, callerMember); err != nil {
		return UpdateMemberStatusOutput{}, err
	}

	allMembers, err := uc.chatMemberRepo.FindByRoom(ctx, roomID)
	if err != nil {
		return UpdateMemberStatusOutput{}, err
	}

	statuses := make([]MemberStatusInfo, 0, len(allMembers))
	for _, m := range allMembers {
		if m.IsDeleted {
			continue
		}
		statuses = append(statuses, MemberStatusInfo{
			MemberID:   int64(m.ID),
			LastReadAt: m.LastReadAt,
		})
	}

	return UpdateMemberStatusOutput{Members: statuses}, nil
}

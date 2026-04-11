package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmessage"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type excludingChatMessageRepository interface {
	FindByRoomExcludingSenders(ctx context.Context, roomID chatroom.ID, excludedSenderIDs []chatmember.ID, limit, offset uint64) ([]*chatmessage.ChatMessage, error)
	FindByRoomBeforeExcludingSenders(ctx context.Context, roomID chatroom.ID, beforeID chatmessage.ID, excludedSenderIDs []chatmember.ID, limit uint64) ([]*chatmessage.ChatMessage, error)
	FindLatestByRoomExcludingSenders(ctx context.Context, roomID chatroom.ID, excludedSenderIDs []chatmember.ID) (*chatmessage.ChatMessage, error)
	CountByRoomAfterExcludingSenders(ctx context.Context, roomID chatroom.ID, since time.Time, excludedSenderIDs []chatmember.ID) (int64, error)
}

func blockedMembersForCaller(
	ctx context.Context,
	friendshipRepo friendship.Repository,
	participantRepo participant.Repository,
	callerUserID shared.UserID,
	callerParticipantID participant.ID,
	members []*chatmember.ChatMember,
) (map[chatmember.ID]struct{}, bool, bool, error) {
	blocked := make(map[chatmember.ID]struct{})
	if friendshipRepo == nil {
		return blocked, false, false, nil
	}

	blockedByPeer := false
	blockedByMe := false
	for _, member := range members {
		if member.IsDeleted || member.ParticipantID == callerParticipantID {
			continue
		}
		p, err := participantRepo.FindByID(ctx, member.ParticipantID)
		if err != nil || p.UserID == nil {
			continue
		}
		rows, err := friendshipRepo.FindBetweenUsers(ctx, callerUserID, *p.UserID)
		if err != nil {
			if errors.Is(err, friendship.ErrFriendshipNotFound) {
				continue
			}
			return nil, false, false, err
		}
		for _, row := range rows {
			if row.Status != friendship.StatusBlocked {
				continue
			}
			blocked[member.ID] = struct{}{}
			if row.UserID == *p.UserID && row.FriendID == callerUserID {
				blockedByPeer = true
			}
			if row.UserID == callerUserID && row.FriendID == *p.UserID {
				blockedByMe = true
			}
			break
		}
	}

	return blocked, blockedByPeer, blockedByMe, nil
}

func blockedSenderList(blocked map[chatmember.ID]struct{}) []chatmember.ID {
	if len(blocked) == 0 {
		return nil
	}
	ids := make([]chatmember.ID, 0, len(blocked))
	for id := range blocked {
		ids = append(ids, id)
	}
	return ids
}

func messageSenderBlocked(msg *chatmessage.ChatMessage, blocked map[chatmember.ID]struct{}) bool {
	_, ok := blocked[msg.SenderID]
	return ok
}

func findByRoomExcludingSenders(
	ctx context.Context,
	repo chatmessage.Repository,
	roomID chatroom.ID,
	blocked map[chatmember.ID]struct{},
	limit, offset uint64,
) ([]*chatmessage.ChatMessage, error) {
	excluded := blockedSenderList(blocked)
	if len(excluded) == 0 {
		return repo.FindByRoom(ctx, roomID, limit, offset)
	}
	if typed, ok := repo.(excludingChatMessageRepository); ok {
		return typed.FindByRoomExcludingSenders(ctx, roomID, excluded, limit, offset)
	}
	msgs, err := repo.FindByRoom(ctx, roomID, limit, offset)
	if err != nil {
		return nil, err
	}
	filtered := make([]*chatmessage.ChatMessage, 0, len(msgs))
	for _, msg := range msgs {
		if !messageSenderBlocked(msg, blocked) {
			filtered = append(filtered, msg)
		}
	}
	return filtered, nil
}

func findByRoomBeforeExcludingSenders(
	ctx context.Context,
	repo chatmessage.Repository,
	roomID chatroom.ID,
	beforeID chatmessage.ID,
	blocked map[chatmember.ID]struct{},
	limit uint64,
) ([]*chatmessage.ChatMessage, error) {
	excluded := blockedSenderList(blocked)
	if len(excluded) == 0 {
		return repo.FindByRoomBefore(ctx, roomID, beforeID, limit)
	}
	if typed, ok := repo.(excludingChatMessageRepository); ok {
		return typed.FindByRoomBeforeExcludingSenders(ctx, roomID, beforeID, excluded, limit)
	}
	msgs, err := repo.FindByRoomBefore(ctx, roomID, beforeID, limit)
	if err != nil {
		return nil, err
	}
	filtered := make([]*chatmessage.ChatMessage, 0, len(msgs))
	for _, msg := range msgs {
		if !messageSenderBlocked(msg, blocked) {
			filtered = append(filtered, msg)
		}
	}
	return filtered, nil
}

func findLatestByRoomExcludingSenders(
	ctx context.Context,
	repo chatmessage.Repository,
	roomID chatroom.ID,
	blocked map[chatmember.ID]struct{},
) (*chatmessage.ChatMessage, error) {
	excluded := blockedSenderList(blocked)
	if len(excluded) == 0 {
		return repo.FindLatestByRoom(ctx, roomID)
	}
	if typed, ok := repo.(excludingChatMessageRepository); ok {
		return typed.FindLatestByRoomExcludingSenders(ctx, roomID, excluded)
	}
	msg, err := repo.FindLatestByRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if messageSenderBlocked(msg, blocked) {
		return nil, chatmessage.ErrNotFound
	}
	return msg, nil
}

func countByRoomAfterExcludingSenders(
	ctx context.Context,
	repo chatmessage.Repository,
	roomID chatroom.ID,
	since time.Time,
	blocked map[chatmember.ID]struct{},
) (int64, error) {
	excluded := blockedSenderList(blocked)
	if len(excluded) == 0 {
		return repo.CountByRoomAfter(ctx, roomID, since)
	}
	if typed, ok := repo.(excludingChatMessageRepository); ok {
		return typed.CountByRoomAfterExcludingSenders(ctx, roomID, since, excluded)
	}
	return repo.CountByRoomAfter(ctx, roomID, since)
}

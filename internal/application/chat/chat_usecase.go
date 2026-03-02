package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/HiroLiang/goat-server/internal/application/shared"
	"github.com/HiroLiang/goat-server/internal/domain/chatgroup"
	"github.com/HiroLiang/goat-server/internal/domain/chatmember"
	"github.com/HiroLiang/goat-server/internal/domain/chatmessage"
	"github.com/HiroLiang/goat-server/internal/domain/participant"
	"github.com/HiroLiang/goat-server/internal/domain/user"
)

const defaultLimit uint64 = 20
const maxLimit uint64 = 50

type UseCase struct {
	participantRepo participant.Repository
	chatGroupRepo   chatgroup.Repository
	chatMemberRepo  chatmember.Repository
	chatMessageRepo chatmessage.Repository
}

func NewUseCase(
	participantRepo participant.Repository,
	chatGroupRepo chatgroup.Repository,
	chatMemberRepo chatmember.Repository,
	chatMessageRepo chatmessage.Repository,
) *UseCase {
	return &UseCase{
		participantRepo: participantRepo,
		chatGroupRepo:   chatGroupRepo,
		chatMemberRepo:  chatMemberRepo,
		chatMessageRepo: chatMessageRepo,
	}
}

// GetMyGroups returns all chat groups the current user belongs to, with unread counts.
func (u *UseCase) GetMyGroups(
	ctx context.Context,
	input shared.UseCaseInput[GetMyGroupsInput],
) (GetMyGroupsOutput, error) {
	userID, err := user.ParseID(input.Base.Auth.UserID)
	if err != nil {
		return GetMyGroupsOutput{}, user.ErrInvalidUser
	}

	p, err := u.participantRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return GetMyGroupsOutput{Groups: []ChatGroupItem{}}, nil
		}
		return GetMyGroupsOutput{}, err
	}

	members, err := u.chatMemberRepo.FindByParticipant(ctx, p.ID)
	if err != nil {
		return GetMyGroupsOutput{}, err
	}

	// Count members per group (for memberCount field)
	groupMemberCounts := make(map[chatgroup.ID]int)
	for _, m := range members {
		groupMemberCounts[m.GroupID]++
	}
	// Actually, we need all members per group, not just the current user's
	// We'll fetch member count per group separately
	items := make([]ChatGroupItem, 0, len(members))

	for _, m := range members {
		group, err := u.chatGroupRepo.FindByID(ctx, m.GroupID)
		if err != nil || group.IsDeleted {
			continue
		}

		// Count all members in this group
		groupMembers, err := u.chatMemberRepo.FindByGroup(ctx, m.GroupID)
		memberCount := 0
		if err == nil {
			memberCount = len(groupMembers)
		}

		// Count unread messages
		var unreadCount int64
		if m.LastReadAt != nil {
			unreadCount, _ = u.chatMessageRepo.CountByGroupAfter(ctx, m.GroupID, *m.LastReadAt)
		} else {
			// Never read: count all messages
			unreadCount, _ = u.chatMessageRepo.CountByGroupAfter(ctx, m.GroupID, time.Time{})
		}

		// Get last message preview
		var lastMsg *LastMessagePreview
		latest, err := u.chatMessageRepo.FindLatestByGroup(ctx, m.GroupID)
		if err == nil {
			senderName := ""
			sender, err := u.participantRepo.FindByID(ctx, latest.SenderID)
			if err == nil {
				senderName = sender.DisplayName
			}
			lastMsg = &LastMessagePreview{
				Content:    latest.Content,
				SenderName: senderName,
				Timestamp:  latest.CreatedAt.UTC().Format(time.RFC3339),
			}
		}

		items = append(items, ChatGroupItem{
			ID:          int64(group.ID),
			Type:        string(group.Type),
			Name:        group.Name,
			Description: group.Description,
			AvatarURL:   group.AvatarURL,
			LastMessage: lastMsg,
			UnreadCount: unreadCount,
			MemberCount: memberCount,
		})
	}

	return GetMyGroupsOutput{Groups: items}, nil
}

// GetGroupMessages returns paginated messages for a group using cursor-based pagination.
func (u *UseCase) GetGroupMessages(
	ctx context.Context,
	input shared.UseCaseInput[GetGroupMessagesInput],
) (GetGroupMessagesOutput, error) {
	userID, err := user.ParseID(input.Base.Auth.UserID)
	if err != nil {
		return GetGroupMessagesOutput{}, user.ErrInvalidUser
	}

	groupID := chatgroup.ID(input.Data.GroupID)

	// Verify the group exists
	group, err := u.chatGroupRepo.FindByID(ctx, groupID)
	if err != nil {
		return GetGroupMessagesOutput{}, chatgroup.ErrNotFound
	}
	if group.IsDeleted {
		return GetGroupMessagesOutput{}, chatgroup.ErrDeleted
	}

	// Verify the current user is a member
	currentParticipant, err := u.participantRepo.FindByUserID(ctx, userID)
	if err != nil {
		return GetGroupMessagesOutput{}, chatgroup.ErrForbidden
	}
	_, err = u.chatMemberRepo.FindByGroupAndParticipant(ctx, groupID, currentParticipant.ID)
	if err != nil {
		return GetGroupMessagesOutput{}, chatgroup.ErrForbidden
	}

	limit := input.Data.Limit
	if limit == 0 {
		limit = defaultLimit
	} else if limit > maxLimit {
		limit = maxLimit
	}

	// Fetch one extra to determine hasMore
	fetchLimit := limit + 1

	var messages []*chatmessage.ChatMessage
	if input.Data.BeforeID != nil {
		messages, err = u.chatMessageRepo.FindByGroupBefore(ctx, groupID, chatmessage.ID(*input.Data.BeforeID), fetchLimit)
	} else {
		// No cursor: fetch the latest messages (descending then reverse)
		raw, fetchErr := u.chatMessageRepo.FindByGroup(ctx, groupID, fetchLimit, 0)
		if fetchErr != nil {
			return GetGroupMessagesOutput{}, fetchErr
		}
		// FindByGroup returns DESC; reverse to ascending
		for i, j := 0, len(raw)-1; i < j; i, j = i+1, j-1 {
			raw[i], raw[j] = raw[j], raw[i]
		}
		messages = raw
		err = nil
	}
	if err != nil {
		return GetGroupMessagesOutput{}, err
	}

	hasMore := false
	if uint64(len(messages)) > limit {
		hasMore = true
		// Remove the ancient message (it's at front after reversal)
		messages = messages[1:]
	}

	// Build participant display name cache
	participantCache := make(map[participant.ID]string)
	avatarCache := make(map[participant.ID]string)

	items := make([]ChatMessageItem, 0, len(messages))
	for _, msg := range messages {
		senderName, ok := participantCache[msg.SenderID]
		senderAvatar := avatarCache[msg.SenderID]
		if !ok {
			sender, err := u.participantRepo.FindByID(ctx, msg.SenderID)
			if err == nil {
				senderName = sender.DisplayName
				senderAvatar = sender.AvatarURL
			}
			participantCache[msg.SenderID] = senderName
			avatarCache[msg.SenderID] = senderAvatar
		}

		var replyToID *int64
		if msg.ReplyToID != nil {
			v := int64(*msg.ReplyToID)
			replyToID = &v
		}

		items = append(items, ChatMessageItem{
			ID:           int64(msg.ID),
			ChatID:       int64(msg.GroupID),
			SenderID:     int64(msg.SenderID),
			SenderName:   senderName,
			SenderAvatar: senderAvatar,
			Content:      msg.Content,
			Type:         msg.Type,
			ReplyToID:    replyToID,
			IsEdited:     msg.IsEdited,
			IsMe:         msg.SenderID == currentParticipant.ID,
			Timestamp:    msg.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	var nextCursor *int64
	if hasMore && len(items) > 0 {
		v := items[0].ID
		nextCursor = &v
	}

	return GetGroupMessagesOutput{
		Messages:   items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// CreateGroup creates a new chat group of the given type or returns an existing one for direct/bot.
func (u *UseCase) CreateGroup(
	ctx context.Context,
	input shared.UseCaseInput[CreateGroupInput],
) (CreateGroupOutput, error) {
	userID, err := user.ParseID(input.Base.Auth.UserID)
	if err != nil {
		return CreateGroupOutput{}, user.ErrInvalidUser
	}

	currentParticipant, err := u.participantRepo.FindByUserID(ctx, userID)
	if err != nil {
		return CreateGroupOutput{}, participant.ErrNotFound
	}

	groupType, err := chatgroup.ParseGroupType(input.Data.Type)
	if err != nil {
		return CreateGroupOutput{}, chatgroup.ErrInvalidGroupType
	}

	switch groupType {
	case chatgroup.Direct:
		return u.createDirectGroup(ctx, userID, currentParticipant, input.Data)
	case chatgroup.Group:
		return u.createRegularGroup(ctx, userID, currentParticipant, input.Data)
	case chatgroup.Bot:
		return u.createBotGroup(ctx, currentParticipant, input.Data)
	default:
		return CreateGroupOutput{}, chatgroup.ErrInvalidGroupType
	}
}

func (u *UseCase) createDirectGroup(
	ctx context.Context,
	userID user.ID,
	currentParticipant *participant.Participant,
	data CreateGroupInput,
) (CreateGroupOutput, error) {
	if len(data.MemberIDs) != 1 {
		return CreateGroupOutput{}, fmt.Errorf("%w: direct group requires exactly 1 member", chatgroup.ErrInvalidGroupType)
	}

	targetParticipant, err := u.participantRepo.FindByID(ctx, participant.ID(data.MemberIDs[0]))
	if err != nil {
		return CreateGroupOutput{}, participant.ErrNotFound
	}
	if !targetParticipant.IsUser() {
		return CreateGroupOutput{}, fmt.Errorf("%w: direct group member must be a user", chatgroup.ErrInvalidGroupType)
	}

	existing, err := u.chatGroupRepo.FindDirectByParticipants(ctx, currentParticipant.ID, targetParticipant.ID)
	if err == nil {
		groupMembers, _ := u.chatMemberRepo.FindByGroup(ctx, existing.ID)
		return CreateGroupOutput{
			Group: ChatGroupItem{
				ID:          int64(existing.ID),
				Type:        string(existing.Type),
				Name:        existing.Name,
				Description: existing.Description,
				AvatarURL:   existing.AvatarURL,
				MemberCount: len(groupMembers),
			},
			IsCreated: false,
		}, nil
	}
	if !errors.Is(err, chatgroup.ErrNotFound) {
		return CreateGroupOutput{}, err
	}

	newGroup := chatgroup.NewDirectGroup(userID)
	if err := u.chatGroupRepo.Create(ctx, newGroup); err != nil {
		return CreateGroupOutput{}, err
	}
	if err := u.chatMemberRepo.Add(ctx, chatmember.NewChatMember(newGroup.ID, currentParticipant.ID, chatmember.Owner)); err != nil {
		return CreateGroupOutput{}, err
	}
	if err := u.chatMemberRepo.Add(ctx, chatmember.NewChatMember(newGroup.ID, targetParticipant.ID, chatmember.Member)); err != nil {
		return CreateGroupOutput{}, err
	}

	return CreateGroupOutput{
		Group: ChatGroupItem{
			ID:          int64(newGroup.ID),
			Type:        string(newGroup.Type),
			Name:        newGroup.Name,
			Description: newGroup.Description,
			AvatarURL:   newGroup.AvatarURL,
			MemberCount: 2,
		},
		IsCreated: true,
	}, nil
}

func (u *UseCase) createRegularGroup(
	ctx context.Context,
	userID user.ID,
	currentParticipant *participant.Participant,
	data CreateGroupInput,
) (CreateGroupOutput, error) {
	if data.Name == "" {
		return CreateGroupOutput{}, fmt.Errorf("%w: group name is required", chatgroup.ErrInvalidGroupType)
	}

	maxMembers := 100
	if data.MaxMembers != nil {
		maxMembers = *data.MaxMembers
	}

	newGroup := chatgroup.NewGroup(data.Name, data.Description, maxMembers, userID)
	if err := u.chatGroupRepo.Create(ctx, newGroup); err != nil {
		return CreateGroupOutput{}, err
	}
	if err := u.chatMemberRepo.Add(ctx, chatmember.NewChatMember(newGroup.ID, currentParticipant.ID, chatmember.Owner)); err != nil {
		return CreateGroupOutput{}, err
	}

	memberCount := 1
	for _, memberID := range data.MemberIDs {
		if participant.ID(memberID) == currentParticipant.ID {
			continue
		}
		p, err := u.participantRepo.FindByID(ctx, participant.ID(memberID))
		if err != nil {
			return CreateGroupOutput{}, participant.ErrNotFound
		}
		if err := u.chatMemberRepo.Add(ctx, chatmember.NewChatMember(newGroup.ID, p.ID, chatmember.Member)); err != nil {
			return CreateGroupOutput{}, err
		}
		memberCount++
	}

	return CreateGroupOutput{
		Group: ChatGroupItem{
			ID:          int64(newGroup.ID),
			Type:        string(newGroup.Type),
			Name:        newGroup.Name,
			Description: newGroup.Description,
			AvatarURL:   newGroup.AvatarURL,
			MemberCount: memberCount,
		},
		IsCreated: true,
	}, nil
}

func (u *UseCase) createBotGroup(
	ctx context.Context,
	currentParticipant *participant.Participant,
	data CreateGroupInput,
) (CreateGroupOutput, error) {
	if len(data.MemberIDs) != 1 {
		return CreateGroupOutput{}, fmt.Errorf("%w: bot group requires exactly 1 agent member", chatgroup.ErrInvalidGroupType)
	}

	agentParticipant, err := u.participantRepo.FindByID(ctx, participant.ID(data.MemberIDs[0]))
	if err != nil {
		return CreateGroupOutput{}, participant.ErrNotFound
	}
	if !agentParticipant.IsAgent() {
		return CreateGroupOutput{}, fmt.Errorf("%w: bot group member must be an agent", chatgroup.ErrInvalidGroupType)
	}

	// Search for an existing bot group containing both participants
	members, err := u.chatMemberRepo.FindByParticipant(ctx, currentParticipant.ID)
	if err != nil {
		return CreateGroupOutput{}, err
	}
	for _, m := range members {
		group, err := u.chatGroupRepo.FindByID(ctx, m.GroupID)
		if err != nil || group.IsDeleted || group.Type != chatgroup.Bot {
			continue
		}
		_, err = u.chatMemberRepo.FindByGroupAndParticipant(ctx, m.GroupID, agentParticipant.ID)
		if err == nil {
			groupMembers, _ := u.chatMemberRepo.FindByGroup(ctx, m.GroupID)
			return CreateGroupOutput{
				Group: ChatGroupItem{
					ID:          int64(group.ID),
					Type:        string(group.Type),
					Name:        group.Name,
					Description: group.Description,
					AvatarURL:   group.AvatarURL,
					MemberCount: len(groupMembers),
				},
				IsCreated: false,
			}, nil
		}
	}

	newGroup := chatgroup.NewBotGroup(data.Name, data.Description)
	if err := u.chatGroupRepo.Create(ctx, newGroup); err != nil {
		return CreateGroupOutput{}, err
	}
	if err := u.chatMemberRepo.Add(ctx, chatmember.NewChatMember(newGroup.ID, currentParticipant.ID, chatmember.Member)); err != nil {
		return CreateGroupOutput{}, err
	}
	if err := u.chatMemberRepo.Add(ctx, chatmember.NewChatMember(newGroup.ID, agentParticipant.ID, chatmember.Member)); err != nil {
		return CreateGroupOutput{}, err
	}

	return CreateGroupOutput{
		Group: ChatGroupItem{
			ID:          int64(newGroup.ID),
			Type:        string(newGroup.Type),
			Name:        newGroup.Name,
			Description: newGroup.Description,
			AvatarURL:   newGroup.AvatarURL,
			MemberCount: 2,
		},
		IsCreated: true,
	}, nil
}

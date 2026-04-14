package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	accountfeatures "github.com/HiroLiang/tentserv-chat-server/features/account"
	bddsupport "github.com/HiroLiang/tentserv-chat-server/features/support"
	chatPort "github.com/HiroLiang/tentserv-chat-server/internal/application/chat/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/cucumber/godog"
)

type steps struct {
	*bddsupport.APITestContext
	deps             *Deps
	accountBDD       *accountfeatures.Deps
	start            time.Time
	lastDirectRoomID chatroom.ID
}

type roomSummaryResponse struct {
	RoomID                 int64      `json:"room_id"`
	RoomType               string     `json:"room_type"`
	DisplayName            string     `json:"display_name"`
	AvatarURL              *string    `json:"avatar_url"`
	PeerUserID             *int64     `json:"peer_user_id"`
	PresenceStatus         *string    `json:"presence_status"`
	LastSeenAt             *time.Time `json:"last_seen_at"`
	LatestMessage          *string    `json:"latest_message"`
	LatestMessageCreatedAt *time.Time `json:"latest_message_created_at"`
	LatestMessageSender    *int64     `json:"latest_message_sender_id"`
	UnreadCount            int64      `json:"unread_count"`
	BlockedByPeer          bool       `json:"blocked_by_peer"`
	BlockedByMe            bool       `json:"blocked_by_me"`
}

type getUserRoomsResponse struct {
	Direct  []roomSummaryResponse `json:"direct"`
	Group   []roomSummaryResponse `json:"group"`
	Channel []roomSummaryResponse `json:"channel"`
	Bot     []roomSummaryResponse `json:"bot"`
}

type chatMessageResponse struct {
	Content string `json:"content"`
}

type getChatRoomDetailResponse struct {
	Name          string                `json:"name"`
	BlockedByPeer bool                  `json:"blocked_by_peer"`
	BlockedByMe   bool                  `json:"blocked_by_me"`
	Messages      []chatMessageResponse `json:"messages"`
}

type getChatRoomMessagesResponse struct {
	Messages []chatMessageResponse `json:"messages"`
}

type memberStatusInfoResponse struct {
	MemberID   int64      `json:"member_id"`
	LastReadAt *time.Time `json:"last_read_at"`
}

type updateMemberStatusResponse struct {
	Members []memberStatusInfoResponse `json:"members"`
}

func RegisterSteps(ctx *godog.ScenarioContext, apiCtx *bddsupport.APITestContext, deps *Deps, accountBDD *accountfeatures.Deps) {
	s := &steps{APITestContext: apiCtx, deps: deps, accountBDD: accountBDD}

	ctx.Step(`^chat room summary state is clean$`, s.chatRoomSummaryStateIsClean)
	ctx.Step(`^the logged in chat user exists as "([^"]*)"$`, s.theLoggedInChatUserExistsAs)
	ctx.Step(`^a direct chat peer "([^"]*)" exists with id (\d+) and avatar "([^"]*)"$`, s.aDirectChatPeerExistsWithIDAndAvatar)
	ctx.Step(`^I have a direct chat room with user (\d+)$`, s.iHaveADirectChatRoomWithUser)
	ctx.Step(`^I have a group chat room "([^"]*)" with users (\d+) and (\d+)$`, s.iHaveAGroupChatRoomWithUsers)
	ctx.Step(`^the direct chat room has latest message "([^"]*)" from user (\d+)$`, s.theDirectChatRoomHasLatestMessageFromUser)
	ctx.Step(`^the direct chat room has latest message "([^"]*)" from the logged in user$`, s.theDirectChatRoomHasLatestMessageFromTheLoggedInUser)
	ctx.Step(`^the last chat room has latest message "([^"]*)" from user (\d+)$`, s.theLastChatRoomHasLatestMessageFromUser)
	ctx.Step(`^the chat presence for user (\d+) is "([^"]*)"$`, s.theChatPresenceForUserIs)
	ctx.Step(`^the chat presence for user (\d+) is "([^"]*)" with last seen "([^"]*)"$`, s.theChatPresenceForUserIsWithLastSeen)
	ctx.Step(`^user (\d+) has blocked the logged in chat user$`, s.userHasBlockedTheLoggedInChatUser)
	ctx.Step(`^the logged in chat user has blocked user (\d+)$`, s.theLoggedInChatUserHasBlockedUser)
	ctx.Step(`^the direct chat room is deleted$`, s.theDirectChatRoomIsDeleted)
	ctx.Step(`^I request my chat rooms$`, s.iRequestMyChatRooms)
	ctx.Step(`^I request chat room detail for the last direct room$`, s.iRequestChatRoomDetailForTheLastDirectRoom)
	ctx.Step(`^I request chat room messages for the last direct room$`, s.iRequestChatRoomMessagesForTheLastDirectRoom)
	ctx.Step(`^I request chat room messages for the last chat room$`, s.iRequestChatRoomMessagesForTheLastChatRoom)
	ctx.Step(`^I mark the last direct room as read$`, s.iMarkTheLastDirectRoomAsRead)
	ctx.Step(`^I send a text message "([^"]*)" to the last direct room$`, s.iSendATextMessageToTheLastDirectRoom)
	ctx.Step(`^the direct chat rooms response should include "([^"]*)" with avatar "([^"]*)", latest message "([^"]*)", latest message sender member id from user (\d+), and latest message created_at$`, s.theDirectChatRoomsResponseShouldIncludeLatestMessageSender)
	ctx.Step(`^the direct chat rooms response should include "([^"]*)" with peer user id (\d+) and presence "([^"]*)"$`, s.theDirectChatRoomsResponseShouldIncludePresence)
	ctx.Step(`^the direct chat rooms response should include "([^"]*)" with peer user id (\d+), presence "([^"]*)", and last seen "([^"]*)"$`, s.theDirectChatRoomsResponseShouldIncludePresenceWithLastSeen)
	ctx.Step(`^the direct chat rooms response should include "([^"]*)" with unread count (\d+)$`, s.theDirectChatRoomsResponseShouldIncludeUnreadCount)
	ctx.Step(`^the direct chat rooms response should include "([^"]*)" marked blocked by peer with latest message "([^"]*)"$`, s.theDirectChatRoomsResponseShouldIncludeMarkedBlockedByPeerWithLatestMessage)
	ctx.Step(`^the direct chat rooms response should include "([^"]*)" marked blocked by me with latest message "([^"]*)"$`, s.theDirectChatRoomsResponseShouldIncludeMarkedBlockedByMeWithLatestMessage)
	ctx.Step(`^the direct chat rooms response should not include "([^"]*)"$`, s.theDirectChatRoomsResponseShouldNotInclude)
	ctx.Step(`^the member status response should include the logged in member with last_read_at$`, s.theMemberStatusResponseShouldIncludeTheLoggedInMemberWithLastReadAt)
	ctx.Step(`^the chat room detail response should be marked blocked by peer$`, s.theChatRoomDetailResponseShouldBeMarkedBlockedByPeer)
	ctx.Step(`^the chat room detail response should be marked blocked by me$`, s.theChatRoomDetailResponseShouldBeMarkedBlockedByMe)
	ctx.Step(`^the chat room detail response should have name "([^"]*)"$`, s.theChatRoomDetailResponseShouldHaveName)
	ctx.Step(`^the chat room response should include message "([^"]*)"$`, s.theChatRoomResponseShouldIncludeMessage)
	ctx.Step(`^the chat room response should not include message "([^"]*)"$`, s.theChatRoomResponseShouldNotIncludeMessage)
}

func (s *steps) chatRoomSummaryStateIsClean() error {
	s.start = time.Now()
	fmt.Println("Given: chat room summary state is clean")
	fmt.Println("Input: reset_requested=true")
	fmt.Println("Action: reset in-memory chat summary dependencies")
	s.deps.Reset()
	s.lastDirectRoomID = 0
	fmt.Println("Output: chat_rooms=0 chat_members=0 chat_messages=0")
	fmt.Println("Mutation: chat summary repositories reset")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) theLoggedInChatUserExistsAs(name string) error {
	s.start = time.Now()
	userID := s.accountBDD.LastSessionUserID()
	if userID == 0 {
		return fmt.Errorf("no logged in user available")
	}
	fmt.Println("Given: the logged in user has a chat participant")
	fmt.Printf("Input: user_id=%d name=%s\n", userID, name)
	fmt.Println("Action: seed chat user and participant")
	s.deps.SeedUser(userID, name, "")
	fmt.Println("Output: user_seeded=true participant_seeded=true")
	fmt.Println("Mutation: chat user repository expanded")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) aDirectChatPeerExistsWithIDAndAvatar(name string, userID int64, avatar string) error {
	s.start = time.Now()
	fmt.Println("Given: a direct chat peer exists")
	fmt.Printf("Input: user_id=%d name=%s avatar=%s\n", userID, name, avatar)
	fmt.Println("Action: seed peer user and participant")
	s.deps.SeedUser(shared.UserID(userID), name, avatar)
	fmt.Println("Output: user_seeded=true participant_seeded=true")
	fmt.Println("Mutation: chat user repository expanded")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iHaveADirectChatRoomWithUser(userID int64) error {
	s.start = time.Now()
	currentUserID := s.accountBDD.LastSessionUserID()
	if currentUserID == 0 {
		return fmt.Errorf("no logged in user available")
	}
	fmt.Println("Given: the logged in user and peer should share a direct room")
	fmt.Printf("Input: current_user_id=%d peer_user_id=%d\n", currentUserID, userID)
	fmt.Println("Action: create direct chat room and owner members")
	s.lastDirectRoomID = s.deps.CreateDirectRoomBetweenUsers(currentUserID, shared.UserID(userID))
	fmt.Printf("Output: room_id=%d\n", s.lastDirectRoomID)
	fmt.Println("Mutation: direct room and members inserted")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	if s.lastDirectRoomID == 0 {
		return fmt.Errorf("failed to create direct room")
	}
	return nil
}

func (s *steps) iHaveAGroupChatRoomWithUsers(name string, userID1, userID2 int64) error {
	s.start = time.Now()
	currentUserID := s.accountBDD.LastSessionUserID()
	if currentUserID == 0 {
		return fmt.Errorf("no logged in user available")
	}
	fmt.Println("Given: the logged in user and peers should share a group room")
	fmt.Printf("Input: current_user_id=%d peer_user_ids=[%d,%d] room_name=%s\n", currentUserID, userID1, userID2, name)
	fmt.Println("Action: create group chat room and members")
	s.lastDirectRoomID = s.deps.CreateGroupRoomWithUsers(name, currentUserID, shared.UserID(userID1), shared.UserID(userID2))
	fmt.Printf("Output: room_id=%d\n", s.lastDirectRoomID)
	fmt.Println("Mutation: group room and members inserted")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	if s.lastDirectRoomID == 0 {
		return fmt.Errorf("failed to create group room")
	}
	return nil
}

func (s *steps) theDirectChatRoomHasLatestMessageFromUser(content string, userID int64) error {
	s.start = time.Now()
	if s.lastDirectRoomID == 0 {
		return fmt.Errorf("no direct room available")
	}
	fmt.Println("Given: a direct room has a latest message")
	fmt.Printf("Input: room_id=%d sender_user_id=%d content_present=%t\n", s.lastDirectRoomID, userID, content != "")
	fmt.Println("Action: seed latest chat message")
	memberID := s.deps.SeedLatestMessage(s.lastDirectRoomID, shared.UserID(userID), content)
	fmt.Printf("Output: sender_member_id=%d\n", memberID)
	fmt.Println("Mutation: chat message inserted")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	if memberID == 0 {
		return fmt.Errorf("failed to resolve sender member id for user %d", userID)
	}
	return nil
}

func (s *steps) theDirectChatRoomHasLatestMessageFromTheLoggedInUser(content string) error {
	userID := int64(s.accountBDD.LastSessionUserID())
	if userID == 0 {
		return fmt.Errorf("no logged in user available")
	}
	return s.theDirectChatRoomHasLatestMessageFromUser(content, userID)
}

func (s *steps) theLastChatRoomHasLatestMessageFromUser(content string, userID int64) error {
	return s.theDirectChatRoomHasLatestMessageFromUser(content, userID)
}

func (s *steps) theChatPresenceForUserIs(userID int64, status string) error {
	return s.setChatPresence(userID, status, "")
}

func (s *steps) theChatPresenceForUserIsWithLastSeen(userID int64, status, lastSeen string) error {
	return s.setChatPresence(userID, status, lastSeen)
}

func (s *steps) setChatPresence(userID int64, status, lastSeen string) error {
	s.start = time.Now()
	fmt.Println("Given: the direct chat peer has a presence snapshot")
	fmt.Printf("Input: user_id=%d status=%s last_seen=%s\n", userID, status, lastSeen)
	fmt.Println("Action: seed fake presence snapshot for chat room summary")

	snapshot := chatPort.PresenceSnapshot{Status: chatPort.PresenceStatus(status)}
	if lastSeen != "" {
		parsed, err := time.Parse(time.RFC3339, lastSeen)
		if err != nil {
			return err
		}
		snapshot.LastSeenAt = &parsed
	}
	s.deps.SetUserPresence(shared.UserID(userID), snapshot)

	fmt.Printf("Output: presence_seeded=true status=%s\n", status)
	fmt.Println("Mutation: fake presence snapshot stored")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) userHasBlockedTheLoggedInChatUser(userID int64) error {
	s.start = time.Now()
	currentUserID := s.accountBDD.LastSessionUserID()
	if currentUserID == 0 {
		return fmt.Errorf("no logged in user available")
	}
	fmt.Println("Given: a peer has blocked the logged in chat user")
	fmt.Printf("Input: blocker_user_id=%d blocked_user_id=%d\n", userID, currentUserID)
	fmt.Println("Action: seed blocked friendship relationship")
	s.deps.SeedBlockedFriendship(shared.UserID(userID), currentUserID)
	fmt.Println("Output: blocked_relationship_seeded=true")
	fmt.Println("Mutation: friendship repository expanded")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) theLoggedInChatUserHasBlockedUser(userID int64) error {
	s.start = time.Now()
	currentUserID := s.accountBDD.LastSessionUserID()
	if currentUserID == 0 {
		return fmt.Errorf("no logged in user available")
	}
	fmt.Println("Given: the logged in chat user has blocked a peer")
	fmt.Printf("Input: blocker_user_id=%d blocked_user_id=%d\n", currentUserID, userID)
	fmt.Println("Action: seed blocked friendship relationship")
	s.deps.SeedBlockedFriendship(currentUserID, shared.UserID(userID))
	fmt.Println("Output: blocked_relationship_seeded=true")
	fmt.Println("Mutation: friendship repository expanded")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) theDirectChatRoomIsDeleted() error {
	s.start = time.Now()
	if s.lastDirectRoomID == 0 {
		return fmt.Errorf("no direct room available")
	}
	fmt.Println("Given: the direct chat room should be soft-deleted")
	fmt.Printf("Input: room_id=%d\n", s.lastDirectRoomID)
	fmt.Println("Action: mark direct room and its members deleted in the chat repository")
	s.deps.MarkRoomDeleted(s.lastDirectRoomID)
	fmt.Printf("Output: room_deleted=true room_id=%d\n", s.lastDirectRoomID)
	fmt.Println("Mutation: direct room and room members soft-deleted")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iRequestMyChatRooms() error {
	s.start = time.Now()
	token := s.accountBDD.LastAccessToken()
	deviceID := s.accountBDD.LastSessionDeviceID().String()
	if token == "" {
		return fmt.Errorf("no login access token available")
	}
	fmt.Println("Given: an authenticated user requests chat room summaries")
	fmt.Printf("Input: token_present=%t device_id=%s\n", token != "", deviceID)
	fmt.Println("Action: GET /api/chat/rooms")
	err := s.DoRequestWithHeaders(http.MethodGet, "/api/chat/rooms", map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
		"X-Device-ID":   deviceID,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iRequestChatRoomDetailForTheLastDirectRoom() error {
	s.start = time.Now()
	if s.lastDirectRoomID == 0 {
		return fmt.Errorf("no direct room available")
	}
	token := s.accountBDD.LastAccessToken()
	deviceID := s.accountBDD.LastSessionDeviceID().String()
	if token == "" {
		return fmt.Errorf("no login access token available")
	}
	fmt.Println("Given: an authenticated user requests deleted chat room detail")
	fmt.Printf("Input: token_present=%t device_id=%s room_id=%d\n", token != "", deviceID, s.lastDirectRoomID)
	fmt.Println("Action: GET /api/chat/room/{room_id}")
	err := s.DoRequestWithHeaders(http.MethodGet, fmt.Sprintf("/api/chat/room/%d", s.lastDirectRoomID), map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
		"X-Device-ID":   deviceID,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iRequestChatRoomMessagesForTheLastDirectRoom() error {
	s.start = time.Now()
	if s.lastDirectRoomID == 0 {
		return fmt.Errorf("no direct room available")
	}
	token := s.accountBDD.LastAccessToken()
	deviceID := s.accountBDD.LastSessionDeviceID().String()
	if token == "" {
		return fmt.Errorf("no login access token available")
	}
	fmt.Println("Given: an authenticated user requests deleted chat room messages")
	fmt.Printf("Input: token_present=%t device_id=%s room_id=%d\n", token != "", deviceID, s.lastDirectRoomID)
	fmt.Println("Action: GET /api/chat/room/{room_id}/messages")
	err := s.DoRequestWithHeaders(http.MethodGet, fmt.Sprintf("/api/chat/room/%d/messages", s.lastDirectRoomID), map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
		"X-Device-ID":   deviceID,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iRequestChatRoomMessagesForTheLastChatRoom() error {
	return s.iRequestChatRoomMessagesForTheLastDirectRoom()
}

func (s *steps) iMarkTheLastDirectRoomAsRead() error {
	s.start = time.Now()
	if s.lastDirectRoomID == 0 {
		return fmt.Errorf("no direct room available")
	}
	token := s.accountBDD.LastAccessToken()
	deviceID := s.accountBDD.LastSessionDeviceID().String()
	if token == "" {
		return fmt.Errorf("no login access token available")
	}
	fmt.Println("Given: an authenticated user marks the direct room as read")
	fmt.Printf("Input: token_present=%t device_id=%s room_id=%d\n", token != "", deviceID, s.lastDirectRoomID)
	fmt.Println("Action: PATCH /api/chat/room/{room_id}/member/status")
	err := s.DoRequestWithHeaders(http.MethodPatch, fmt.Sprintf("/api/chat/room/%d/member/status", s.lastDirectRoomID), map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
		"X-Device-ID":   deviceID,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Println("Mutation: member last_read_at should update for the caller")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iSendATextMessageToTheLastDirectRoom(content string) error {
	s.start = time.Now()
	if s.lastDirectRoomID == 0 {
		return fmt.Errorf("no direct room available")
	}
	token := s.accountBDD.LastAccessToken()
	deviceID := s.accountBDD.LastSessionDeviceID().String()
	if token == "" {
		return fmt.Errorf("no login access token available")
	}
	fmt.Println("Given: an authenticated user sends a text message to a deleted chat room")
	fmt.Printf("Input: token_present=%t device_id=%s room_id=%d content_present=%t\n",
		token != "", deviceID, s.lastDirectRoomID, content != "")
	fmt.Println("Action: POST /api/chat/room/{room_id}/messages")
	body, err := json.Marshal(map[string]any{
		"content":            content,
		"type":               "text",
		"sender_key_version": 1,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/chat/room/%d/messages", s.BaseURL, s.lastDirectRoomID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("X-Device-ID", deviceID)
	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	s.Response = resp
	s.ResponseBody, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Println("Mutation: no message created for deleted room")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) theDirectChatRoomsResponseShouldIncludeLatestMessageSender(name, avatar, latestMessage string, userID int64) error {
	start := time.Now()
	fmt.Println("Given: chat room response should expose direct room summary metadata")
	fmt.Printf("Input: expected_name=%s expected_avatar=%s expected_message_present=%t sender_user_id=%d latest_message_created_at_present=true\n",
		name, avatar, latestMessage != "", userID)
	fmt.Println("Action: decode chat rooms response")

	var body getUserRoomsResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	expectedMemberID := int64(s.deps.MemberIDForUser(s.lastDirectRoomID, shared.UserID(userID)))
	for _, room := range body.Direct {
		if room.DisplayName != name || room.AvatarURL == nil || *room.AvatarURL != avatar {
			continue
		}
		messageMatches := room.LatestMessage != nil && *room.LatestMessage == latestMessage
		createdAtMatches := room.LatestMessageCreatedAt != nil
		senderMatches := room.LatestMessageSender != nil && *room.LatestMessageSender == expectedMemberID
		fmt.Printf("Output: matched_room_id=%d message_matches=%t created_at_matches=%t sender_matches=%t expected_sender_member_id=%d\n",
			room.RoomID, messageMatches, createdAtMatches, senderMatches, expectedMemberID)
		fmt.Println("Mutation: none")
		fmt.Printf("Duration: %s\n", time.Since(start))
		if !messageMatches || !createdAtMatches || !senderMatches {
			return fmt.Errorf("expected latest message %q, created_at, and sender member id %d, got %+v", latestMessage, expectedMemberID, room)
		}
		return nil
	}

	return fmt.Errorf("expected direct room response to include %s with avatar %s; body=%s", name, avatar, string(s.ResponseBody))
}

func (s *steps) theDirectChatRoomsResponseShouldIncludeUnreadCount(name string, unreadCount int64) error {
	start := time.Now()
	fmt.Println("Given: chat room response should expose the authoritative unread count")
	fmt.Printf("Input: expected_name=%s expected_unread_count=%d\n", name, unreadCount)
	fmt.Println("Action: decode chat rooms response")

	var body getUserRoomsResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	for _, room := range body.Direct {
		if room.DisplayName != name {
			continue
		}
		fmt.Printf("Output: matched_room_id=%d unread_count=%d\n", room.RoomID, room.UnreadCount)
		fmt.Println("Mutation: none")
		fmt.Printf("Duration: %s\n", time.Since(start))
		if room.UnreadCount != unreadCount {
			return fmt.Errorf("expected direct room %s unread_count=%d, got %+v", name, unreadCount, room)
		}
		return nil
	}
	return fmt.Errorf("expected direct room response to include %s; body=%s", name, string(s.ResponseBody))
}

func (s *steps) theMemberStatusResponseShouldIncludeTheLoggedInMemberWithLastReadAt() error {
	start := time.Now()
	currentUserID := s.accountBDD.LastSessionUserID()
	if currentUserID == 0 {
		return fmt.Errorf("no logged in user available")
	}
	expectedMemberID := int64(s.deps.MemberIDForUser(s.lastDirectRoomID, currentUserID))
	if expectedMemberID == 0 {
		return fmt.Errorf("no member id found for logged in user %d", currentUserID)
	}
	fmt.Println("Given: update member status response should include the caller member")
	fmt.Printf("Input: expected_member_id=%d\n", expectedMemberID)
	fmt.Println("Action: decode update member status response")

	var body updateMemberStatusResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	for _, member := range body.Members {
		if member.MemberID != expectedMemberID {
			continue
		}
		fmt.Printf("Output: matched_member_id=%d last_read_at_present=%t\n", member.MemberID, member.LastReadAt != nil)
		fmt.Println("Mutation: none")
		fmt.Printf("Duration: %s\n", time.Since(start))
		if member.LastReadAt == nil {
			return fmt.Errorf("expected last_read_at for member %d, got %+v", expectedMemberID, member)
		}
		return nil
	}
	return fmt.Errorf("expected member status response to include member %d; body=%s", expectedMemberID, string(s.ResponseBody))
}

func (s *steps) theDirectChatRoomsResponseShouldIncludePresence(name string, peerUserID int64, status string) error {
	start := time.Now()
	fmt.Println("Given: direct room summary should expose peer presence")
	fmt.Printf("Input: expected_name=%s expected_peer_user_id=%d expected_status=%s\n", name, peerUserID, status)
	fmt.Println("Action: decode chat rooms response")

	var body getUserRoomsResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	for _, room := range body.Direct {
		if room.DisplayName != name {
			continue
		}
		peerMatches := room.PeerUserID != nil && *room.PeerUserID == peerUserID
		statusMatches := room.PresenceStatus != nil && *room.PresenceStatus == status
		fmt.Printf("Output: matched_room_id=%d peer_matches=%t status_matches=%t\n", room.RoomID, peerMatches, statusMatches)
		fmt.Println("Mutation: none")
		fmt.Printf("Duration: %s\n", time.Since(start))
		if !peerMatches || !statusMatches {
			return fmt.Errorf("expected peer_user_id=%d and presence=%s, got %+v", peerUserID, status, room)
		}
		return nil
	}
	return fmt.Errorf("expected direct room response to include %s; body=%s", name, string(s.ResponseBody))
}

func (s *steps) theDirectChatRoomsResponseShouldIncludePresenceWithLastSeen(name string, peerUserID int64, status, lastSeen string) error {
	start := time.Now()
	fmt.Println("Given: direct room summary should expose peer offline last seen")
	fmt.Printf("Input: expected_name=%s expected_peer_user_id=%d expected_status=%s expected_last_seen=%s\n", name, peerUserID, status, lastSeen)
	fmt.Println("Action: decode chat rooms response")

	expectedLastSeen, err := time.Parse(time.RFC3339, lastSeen)
	if err != nil {
		return err
	}

	var body getUserRoomsResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	for _, room := range body.Direct {
		if room.DisplayName != name {
			continue
		}
		peerMatches := room.PeerUserID != nil && *room.PeerUserID == peerUserID
		statusMatches := room.PresenceStatus != nil && *room.PresenceStatus == status
		lastSeenMatches := room.LastSeenAt != nil && room.LastSeenAt.Equal(expectedLastSeen)
		fmt.Printf("Output: matched_room_id=%d peer_matches=%t status_matches=%t last_seen_matches=%t\n",
			room.RoomID, peerMatches, statusMatches, lastSeenMatches)
		fmt.Println("Mutation: none")
		fmt.Printf("Duration: %s\n", time.Since(start))
		if !peerMatches || !statusMatches || !lastSeenMatches {
			return fmt.Errorf("expected peer_user_id=%d presence=%s last_seen=%s, got %+v", peerUserID, status, lastSeen, room)
		}
		return nil
	}
	return fmt.Errorf("expected direct room response to include %s; body=%s", name, string(s.ResponseBody))
}

func (s *steps) theDirectChatRoomsResponseShouldIncludeMarkedBlockedByPeerWithLatestMessage(name, latestMessage string) error {
	start := time.Now()
	fmt.Println("Given: direct room summary should stay visible and mark peer block state")
	fmt.Printf("Input: expected_name=%s expected_message=%s\n", name, latestMessage)
	fmt.Println("Action: decode chat rooms response")

	var body getUserRoomsResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	for _, room := range body.Direct {
		if room.DisplayName != name {
			continue
		}
		messageMatches := room.LatestMessage != nil && *room.LatestMessage == latestMessage
		fmt.Printf("Output: matched_room_id=%d blocked_by_peer=%t message_matches=%t\n",
			room.RoomID, room.BlockedByPeer, messageMatches)
		fmt.Println("Mutation: none")
		fmt.Printf("Duration: %s\n", time.Since(start))
		if !room.BlockedByPeer || !messageMatches {
			return fmt.Errorf("expected direct room %s to be blocked_by_peer with latest message %q, got %+v", name, latestMessage, room)
		}
		return nil
	}
	return fmt.Errorf("expected direct room response to include %s; body=%s", name, string(s.ResponseBody))
}

func (s *steps) theDirectChatRoomsResponseShouldIncludeMarkedBlockedByMeWithLatestMessage(name, latestMessage string) error {
	start := time.Now()
	fmt.Println("Given: direct room summary should stay visible and mark caller block state")
	fmt.Printf("Input: expected_name=%s expected_message=%s\n", name, latestMessage)
	fmt.Println("Action: decode chat rooms response")

	var body getUserRoomsResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	for _, room := range body.Direct {
		if room.DisplayName != name {
			continue
		}
		messageMatches := room.LatestMessage != nil && *room.LatestMessage == latestMessage
		fmt.Printf("Output: matched_room_id=%d blocked_by_me=%t message_matches=%t\n",
			room.RoomID, room.BlockedByMe, messageMatches)
		fmt.Println("Mutation: none")
		fmt.Printf("Duration: %s\n", time.Since(start))
		if !room.BlockedByMe || !messageMatches {
			return fmt.Errorf("expected direct room %s to be blocked_by_me with latest message %q, got %+v", name, latestMessage, room)
		}
		return nil
	}
	return fmt.Errorf("expected direct room response to include %s; body=%s", name, string(s.ResponseBody))
}

func (s *steps) theDirectChatRoomsResponseShouldNotInclude(name string) error {
	start := time.Now()
	fmt.Println("Given: chat room response should hide deleted direct rooms")
	fmt.Printf("Input: hidden_display_name=%s\n", name)
	fmt.Println("Action: decode chat rooms response")

	var body getUserRoomsResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	for _, room := range body.Direct {
		if room.DisplayName == name {
			return fmt.Errorf("expected direct rooms response not to include %s; body=%s", name, string(s.ResponseBody))
		}
	}

	fmt.Printf("Output: direct_count=%d hidden=true\n", len(body.Direct))
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (s *steps) theChatRoomDetailResponseShouldBeMarkedBlockedByPeer() error {
	start := time.Now()
	fmt.Println("Given: room detail response should expose peer block state")
	fmt.Println("Input: expected_blocked_by_peer=true")
	fmt.Println("Action: decode room detail response")

	var body getChatRoomDetailResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	fmt.Printf("Output: blocked_by_peer=%t\n", body.BlockedByPeer)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if !body.BlockedByPeer {
		return fmt.Errorf("expected room detail blocked_by_peer=true; body=%s", string(s.ResponseBody))
	}
	return nil
}

func (s *steps) theChatRoomDetailResponseShouldBeMarkedBlockedByMe() error {
	start := time.Now()
	fmt.Println("Given: room detail response should expose caller block state")
	fmt.Println("Input: expected_blocked_by_me=true")
	fmt.Println("Action: decode room detail response")

	var body getChatRoomDetailResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}
	fmt.Printf("Output: blocked_by_me=%t\n", body.BlockedByMe)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if !body.BlockedByMe {
		return fmt.Errorf("expected room detail blocked_by_me=true; body=%s", string(s.ResponseBody))
	}
	return nil
}

func (s *steps) theChatRoomDetailResponseShouldHaveName(expected string) error {
	start := time.Now()
	fmt.Println("Given: chat room detail response should expose a display name")
	fmt.Printf("Input: expected_name=%s\n", expected)
	fmt.Println("Action: decode chat room detail response")

	var body getChatRoomDetailResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}

	fmt.Printf("Output: actual_name=%s\n", body.Name)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if body.Name != expected {
		return fmt.Errorf("expected chat room detail name %q, got %q", expected, body.Name)
	}
	return nil
}

func (s *steps) theChatRoomResponseShouldIncludeMessage(content string) error {
	start := time.Now()
	fmt.Println("Given: room response should include an unblocked message")
	fmt.Printf("Input: expected_content=%s\n", content)
	fmt.Println("Action: decode room response messages")

	messages, err := decodeChatRoomMessages(s.ResponseBody)
	if err != nil {
		return err
	}
	for _, msg := range messages {
		if msg.Content == content {
			fmt.Println("Output: message_found=true")
			fmt.Println("Mutation: none")
			fmt.Printf("Duration: %s\n", time.Since(start))
			return nil
		}
	}
	return fmt.Errorf("expected room response to include message %q; body=%s", content, string(s.ResponseBody))
}

func (s *steps) theChatRoomResponseShouldNotIncludeMessage(content string) error {
	start := time.Now()
	fmt.Println("Given: room response should exclude blocked sender messages")
	fmt.Printf("Input: hidden_content=%s\n", content)
	fmt.Println("Action: decode room response messages")

	messages, err := decodeChatRoomMessages(s.ResponseBody)
	if err != nil {
		return err
	}
	for _, msg := range messages {
		if msg.Content == content {
			return fmt.Errorf("expected room response not to include message %q; body=%s", content, string(s.ResponseBody))
		}
	}
	fmt.Println("Output: message_hidden=true")
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func decodeChatRoomMessages(body []byte) ([]chatMessageResponse, error) {
	var detail getChatRoomDetailResponse
	if err := json.Unmarshal(body, &detail); err == nil && detail.Messages != nil {
		return detail.Messages, nil
	}
	var messages getChatRoomMessagesResponse
	if err := json.Unmarshal(body, &messages); err != nil {
		return nil, err
	}
	return messages.Messages, nil
}

package chat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	accountfeatures "github.com/HiroLiang/tentserv-chat-server/features/account"
	bddsupport "github.com/HiroLiang/tentserv-chat-server/features/support"
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
	RoomID              int64   `json:"room_id"`
	RoomType            string  `json:"room_type"`
	DisplayName         string  `json:"display_name"`
	AvatarURL           *string `json:"avatar_url"`
	LatestMessage       *string `json:"latest_message"`
	LatestMessageSender *int64  `json:"latest_message_sender_id"`
	UnreadCount         int64   `json:"unread_count"`
}

type getUserRoomsResponse struct {
	Direct  []roomSummaryResponse `json:"direct"`
	Group   []roomSummaryResponse `json:"group"`
	Channel []roomSummaryResponse `json:"channel"`
	Bot     []roomSummaryResponse `json:"bot"`
}

func RegisterSteps(ctx *godog.ScenarioContext, apiCtx *bddsupport.APITestContext, deps *Deps, accountBDD *accountfeatures.Deps) {
	s := &steps{APITestContext: apiCtx, deps: deps, accountBDD: accountBDD}

	ctx.Step(`^chat room summary state is clean$`, s.chatRoomSummaryStateIsClean)
	ctx.Step(`^the logged in chat user exists as "([^"]*)"$`, s.theLoggedInChatUserExistsAs)
	ctx.Step(`^a direct chat peer "([^"]*)" exists with id (\d+) and avatar "([^"]*)"$`, s.aDirectChatPeerExistsWithIDAndAvatar)
	ctx.Step(`^I have a direct chat room with user (\d+)$`, s.iHaveADirectChatRoomWithUser)
	ctx.Step(`^the direct chat room has latest message "([^"]*)" from user (\d+)$`, s.theDirectChatRoomHasLatestMessageFromUser)
	ctx.Step(`^the direct chat room is deleted$`, s.theDirectChatRoomIsDeleted)
	ctx.Step(`^I request my chat rooms$`, s.iRequestMyChatRooms)
	ctx.Step(`^I request chat room detail for the last direct room$`, s.iRequestChatRoomDetailForTheLastDirectRoom)
	ctx.Step(`^I request chat room messages for the last direct room$`, s.iRequestChatRoomMessagesForTheLastDirectRoom)
	ctx.Step(`^I send a text message "([^"]*)" to the last direct room$`, s.iSendATextMessageToTheLastDirectRoom)
	ctx.Step(`^the direct chat rooms response should include "([^"]*)" with avatar "([^"]*)", latest message "([^"]*)", and latest message sender member id from user (\d+)$`, s.theDirectChatRoomsResponseShouldIncludeLatestMessageSender)
	ctx.Step(`^the direct chat rooms response should not include "([^"]*)"$`, s.theDirectChatRoomsResponseShouldNotInclude)
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
	err := s.DoJSONRequestWithHeaders(http.MethodPost, fmt.Sprintf("/api/chat/room/%d/messages", s.lastDirectRoomID), map[string]string{
		"content": content,
		"type":    "text",
	}, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
		"X-Device-ID":   deviceID,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Println("Mutation: no message created for deleted room")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) theDirectChatRoomsResponseShouldIncludeLatestMessageSender(name, avatar, latestMessage string, userID int64) error {
	start := time.Now()
	fmt.Println("Given: chat room response should expose direct room summary metadata")
	fmt.Printf("Input: expected_name=%s expected_avatar=%s expected_message_present=%t sender_user_id=%d\n",
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
		senderMatches := room.LatestMessageSender != nil && *room.LatestMessageSender == expectedMemberID
		fmt.Printf("Output: matched_room_id=%d message_matches=%t sender_matches=%t expected_sender_member_id=%d\n",
			room.RoomID, messageMatches, senderMatches, expectedMemberID)
		fmt.Println("Mutation: none")
		fmt.Printf("Duration: %s\n", time.Since(start))
		if !messageMatches || !senderMatches {
			return fmt.Errorf("expected latest message %q and sender member id %d, got %+v", latestMessage, expectedMemberID, room)
		}
		return nil
	}

	return fmt.Errorf("expected direct room response to include %s with avatar %s; body=%s", name, avatar, string(s.ResponseBody))
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

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	accountfeatures "github.com/HiroLiang/tentserv-chat-server/features/account"
	friendshipfeatures "github.com/HiroLiang/tentserv-chat-server/features/friendship"
	bddsupport "github.com/HiroLiang/tentserv-chat-server/features/support"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/cucumber/godog"
)

type steps struct {
	*bddsupport.APITestContext
	accountBDD    *accountfeatures.Deps
	friendshipBDD *friendshipfeatures.Deps
}

type createRoomResponse struct {
	ID             int64  `json:"id"`
	Type           string `json:"type"`
	AlreadyExisted bool   `json:"already_existed"`
}

func RegisterSteps(ctx *godog.ScenarioContext, apiCtx *bddsupport.APITestContext, accountBDD *accountfeatures.Deps, friendshipBDD *friendshipfeatures.Deps) {
	s := &steps{APITestContext: apiCtx, accountBDD: accountBDD, friendshipBDD: friendshipBDD}

	ctx.Step(`^I create a direct chat room for user (\d+) named "([^"]*)" through the chat API$`, s.iCreateADirectChatRoomForUserNamedThroughTheChatAPI)
	ctx.Step(`^the direct room create response should reuse the existing direct room between the logged in user and user (\d+)$`, s.theDirectRoomCreateResponseShouldReuseTheExistingDirectRoomBetweenTheLoggedInUserAndUser)
}

func (s *steps) iCreateADirectChatRoomForUserNamedThroughTheChatAPI(userID int64, name string) error {
	start := time.Now()
	accessToken := s.accountBDD.LastAccessToken()
	deviceID := s.accountBDD.LastSessionDeviceID().String()
	if accessToken == "" {
		return fmt.Errorf("no login access token available for chat room creation")
	}
	if deviceID == "" {
		return fmt.Errorf("no login device id available for chat room creation")
	}

	fmt.Println("Given: an authenticated user wants to open a direct room through the chat API")
	fmt.Printf("Input: target_user_id=%d requested_name=%s token_present=%t device_id=%s\n", userID, name, accessToken != "", deviceID)
	fmt.Println("Action: POST /api/chat/room")

	payload := map[string]any{
		"type":       "direct",
		"name":       name,
		"member_ids": []int64{userID},
	}
	if err := s.doJSONRequestWithHeaders(http.MethodPost, "/api/chat/room", payload, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", accessToken),
		"X-Device-ID":   deviceID,
	}); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (s *steps) theDirectRoomCreateResponseShouldReuseTheExistingDirectRoomBetweenTheLoggedInUserAndUser(userID int64) error {
	start := time.Now()
	currentUserID := s.accountBDD.LastSessionUserID()
	fmt.Println("Given: friendship accept should already have created a direct room")
	fmt.Printf("Input: current_user_id=%d friend_user_id=%d\n", currentUserID, userID)
	fmt.Println("Action: decode create-room response and compare with existing direct room")

	var body createRoomResponse
	if err := json.Unmarshal(s.ResponseBody, &body); err != nil {
		return err
	}

	room, ok := s.friendshipBDD.FindDirectRoomBetweenUsers(currentUserID, shared.UserID(userID))
	if !ok {
		return fmt.Errorf("no existing direct room found between user %d and user %d", currentUserID, userID)
	}

	matches := body.AlreadyExisted && body.Type == "direct" && body.ID == int64(room.ID)
	fmt.Printf("Output: room_id=%d existing_room_id=%d already_existed=%t room_type=%s match=%t\n",
		body.ID, room.ID, body.AlreadyExisted, body.Type, matches)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !matches {
		return fmt.Errorf("expected reused direct room response with id=%d type=direct already_existed=true, got %+v", room.ID, body)
	}
	return nil
}

func (s *steps) doJSONRequestWithHeaders(method, path string, payload any, headers map[string]string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, s.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	s.Response = resp
	s.ResponseBody, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return nil
}

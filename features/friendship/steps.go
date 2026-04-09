package friendship

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	accountfeatures "github.com/HiroLiang/tentserv-chat-server/features/account"
	bddsupport "github.com/HiroLiang/tentserv-chat-server/features/support"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	domainfriendship "github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/cucumber/godog"
)

type steps struct {
	*bddsupport.APITestContext
	deps             *Deps
	accountBDD       *accountfeatures.Deps
	authHeader       string
	start            time.Time
	lastDirectRoomID chatroom.ID
}

type userSearchResponse struct {
	UserID           int64   `json:"user_id"`
	Name             string  `json:"name"`
	Avatar           string  `json:"avatar"`
	Account          string  `json:"account"`
	PublicID         string  `json:"public_id"`
	FriendshipStatus *string `json:"friendship_status,omitempty"`
}

type friendResponse struct {
	FriendshipID int64  `json:"friendship_id"`
	UserID       int64  `json:"user_id"`
	Name         string `json:"name"`
	Avatar       string `json:"avatar"`
	Status       string `json:"status"`
}

type requestResponse struct {
	FriendshipID int64  `json:"friendship_id"`
	UserID       int64  `json:"user_id"`
	Name         string `json:"name"`
	Avatar       string `json:"avatar"`
}

func RegisterSteps(ctx *godog.ScenarioContext, apiCtx *bddsupport.APITestContext, deps *Deps, accountBDD *accountfeatures.Deps) {
	s := &steps{APITestContext: apiCtx, deps: deps, accountBDD: accountBDD}

	ctx.Step(`^friendship state is clean$`, s.friendshipStateIsClean)
	ctx.Step(`^the logged in user exists in the friendship directory as "([^"]*)" with account "([^"]*)" and public id "([^"]*)"$`, s.theLoggedInUserExistsInTheFriendshipDirectory)
	ctx.Step(`^a searchable user "([^"]*)" exists with id (\d+), account "([^"]*)", public id "([^"]*)", and avatar "([^"]*)"$`, s.aSearchableUserExists)
	ctx.Step(`^I have sent a pending friend request to user (\d+)$`, s.iHaveSentAPendingFriendRequestToUser)
	ctx.Step(`^an inbound pending friend request exists from user (\d+)$`, s.anInboundPendingFriendRequestExistsFromUser)
	ctx.Step(`^I am accepted friends with user (\d+)$`, s.iAmAcceptedFriendsWithUser)
	ctx.Step(`^I search friends by name "([^"]*)"$`, s.iSearchFriendsByName)
	ctx.Step(`^search results should include "([^"]*)" with public id "([^"]*)", avatar "([^"]*)", and friendship status "([^"]*)"$`, s.searchResultsShouldIncludeWithPublicIDAvatarAndFriendshipStatus)
	ctx.Step(`^search results should include "([^"]*)" with public id "([^"]*)", avatar "([^"]*)", and no friendship status$`, s.searchResultsShouldIncludeWithPublicIDAvatarAndNoFriendshipStatus)
	ctx.Step(`^search results should not include the logged in user$`, s.searchResultsShouldNotIncludeTheLoggedInUser)
	ctx.Step(`^I apply to user (\d+)$`, s.iApplyToUser)
	ctx.Step(`^I apply with an invalid friend payload$`, s.iApplyWithAnInvalidFriendPayload)
	ctx.Step(`^I apply to the logged in user$`, s.iApplyToTheLoggedInUser)
	ctx.Step(`^friendship rows should contain "([^"]*)" from the logged in user to user (\d+)$`, s.friendshipRowsShouldContainFromTheLoggedInUserToUser)
	ctx.Step(`^I request my friends$`, s.iRequestMyFriends)
	ctx.Step(`^the friends response should include "([^"]*)" with status "([^"]*)"$`, s.theFriendsResponseShouldIncludeWithStatus)
	ctx.Step(`^the friends response should not include user (\d+)$`, s.theFriendsResponseShouldNotIncludeUser)
	ctx.Step(`^I request sent friend requests$`, s.iRequestSentFriendRequests)
	ctx.Step(`^the sent requests response should include "([^"]*)"$`, s.theSentRequestsResponseShouldInclude)
	ctx.Step(`^the sent requests response should not include user (\d+)$`, s.theSentRequestsResponseShouldNotIncludeUser)
	ctx.Step(`^I request incoming friend requests$`, s.iRequestIncomingFriendRequests)
	ctx.Step(`^the incoming requests response should include "([^"]*)"$`, s.theIncomingRequestsResponseShouldInclude)
	ctx.Step(`^the incoming requests response should not include user (\d+)$`, s.theIncomingRequestsResponseShouldNotIncludeUser)
	ctx.Step(`^I request blocked users$`, s.iRequestBlockedUsers)
	ctx.Step(`^the blocked response should include "([^"]*)" with status "([^"]*)"$`, s.theBlockedResponseShouldIncludeWithStatus)
	ctx.Step(`^the blocked response should not include user (\d+)$`, s.theBlockedResponseShouldNotIncludeUser)
	ctx.Step(`^I accept the inbound friend request from user (\d+)$`, s.iAcceptTheInboundFriendRequestFromUser)
	ctx.Step(`^I accept friendship id (\d+)$`, s.iAcceptFriendshipID)
	ctx.Step(`^I reject the inbound friend request from user (\d+)$`, s.iRejectTheInboundFriendRequestFromUser)
	ctx.Step(`^I cancel the sent friend request to user (\d+)$`, s.iCancelTheSentFriendRequestToUser)
	ctx.Step(`^I unfriend user (\d+)$`, s.iUnfriendUser)
	ctx.Step(`^I block user (\d+)$`, s.iBlockUser)
	ctx.Step(`^I unblock user (\d+)$`, s.iUnblockUser)
	ctx.Step(`^mutual friendship rows between the logged in user and user (\d+) should both be "([^"]*)"$`, s.mutualFriendshipRowsBetweenTheLoggedInUserAndUserShouldBothBe)
	ctx.Step(`^friendship rows between the logged in user and user (\d+) should not exist$`, s.friendshipRowsBetweenTheLoggedInUserAndUserShouldNotExist)
	ctx.Step(`^friendship row from user (\d+) to the logged in user should not exist$`, s.friendshipRowFromUserToTheLoggedInUserShouldNotExist)
	ctx.Step(`^a direct chat room should exist between the logged in user and user (\d+)$`, s.aDirectChatRoomShouldExistBetweenLoggedInUserAndUser)
	ctx.Step(`^both members should have role "([^"]*)" in that direct room$`, s.bothMembersShouldHaveRoleInThatDirectRoom)
}

func (s *steps) friendshipStateIsClean() error {
	s.start = time.Now()
	s.deps.Reset()
	pending, accepted, blocked := s.deps.FriendshipCounts()
	fmt.Println("Given: friendship state is clean")
	fmt.Printf("Input: pending=%d accepted=%d blocked=%d users=%d\n", pending, accepted, blocked, len(s.deps.userRepo.usersByID))
	fmt.Println("Action: reset in-memory friendship and searchable user repositories")
	fmt.Printf("Output: pending=%d accepted=%d blocked=%d users=%d\n", pending, accepted, blocked, len(s.deps.userRepo.usersByID))
	fmt.Println("Mutation: friendship repositories reset")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) theLoggedInUserExistsInTheFriendshipDirectory(name, accountName, publicID string) error {
	s.start = time.Now()
	userID := s.accountBDD.LastSessionUserID()
	if userID == 0 {
		return fmt.Errorf("no logged in user available")
	}
	fmt.Println("Given: the logged in user is searchable in the friendship directory")
	fmt.Printf("Input: user_id=%d name=%s account=%s public_id=%s\n", userID, name, accountName, publicID)
	fmt.Println("Action: seed current user into friendship user repository")

	s.deps.SeedUser(userID, name, "", accountName, publicID)

	fmt.Printf("Output: user_seeded=%t\n", true)
	fmt.Println("Mutation: searchable current user added")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) aSearchableUserExists(name string, userID int64, accountName, publicID, avatar string) error {
	s.start = time.Now()
	fmt.Println("Given: another searchable user exists")
	fmt.Printf("Input: user_id=%d name=%s account=%s public_id=%s avatar=%s\n", userID, name, accountName, publicID, avatar)
	fmt.Println("Action: seed searchable user")

	s.deps.SeedUser(shared.UserID(userID), name, avatar, accountName, publicID)

	fmt.Printf("Output: user_seeded=%t\n", true)
	fmt.Println("Mutation: searchable directory expanded")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iHaveSentAPendingFriendRequestToUser(userID int64) error {
	return s.seedDirectedFriendship(shared.UserID(userID), domainfriendship.StatusPending, "outbound pending request")
}

func (s *steps) anInboundPendingFriendRequestExistsFromUser(userID int64) error {
	return s.seedReverseFriendship(shared.UserID(userID), domainfriendship.StatusPending, "inbound pending request")
}

func (s *steps) iAmAcceptedFriendsWithUser(userID int64) error {
	currentUserID := s.accountBDD.LastSessionUserID()
	if currentUserID == 0 {
		return fmt.Errorf("no logged in user available")
	}
	s.start = time.Now()
	fmt.Println("Given: the logged in user is already accepted friends with another user")
	fmt.Printf("Input: current_user_id=%d other_user_id=%d\n", currentUserID, userID)
	fmt.Println("Action: seed mutual accepted directed friendships")

	s.deps.SeedFriendship(currentUserID, shared.UserID(userID), domainfriendship.StatusAccepted)
	s.deps.SeedFriendship(shared.UserID(userID), currentUserID, domainfriendship.StatusAccepted)

	pending, accepted, blocked := s.deps.FriendshipCounts()
	fmt.Printf("Output: pending=%d accepted=%d blocked=%d\n", pending, accepted, blocked)
	fmt.Println("Mutation: accepted rows created in both directions")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) seedDirectedFriendship(otherUserID shared.UserID, status domainfriendship.Status, label string) error {
	currentUserID := s.accountBDD.LastSessionUserID()
	if currentUserID == 0 {
		return fmt.Errorf("no logged in user available")
	}
	s.start = time.Now()
	fmt.Printf("Given: a %s exists\n", label)
	fmt.Printf("Input: from_user_id=%d to_user_id=%d status=%s\n", currentUserID, otherUserID, status)
	fmt.Println("Action: seed directed friendship row")

	s.deps.SeedFriendship(currentUserID, otherUserID, status)

	pending, accepted, blocked := s.deps.FriendshipCounts()
	fmt.Printf("Output: pending=%d accepted=%d blocked=%d\n", pending, accepted, blocked)
	fmt.Println("Mutation: friendship row added")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) seedReverseFriendship(otherUserID shared.UserID, status domainfriendship.Status, label string) error {
	currentUserID := s.accountBDD.LastSessionUserID()
	if currentUserID == 0 {
		return fmt.Errorf("no logged in user available")
	}
	s.start = time.Now()
	fmt.Printf("Given: a %s exists\n", label)
	fmt.Printf("Input: from_user_id=%d to_user_id=%d status=%s\n", otherUserID, currentUserID, status)
	fmt.Println("Action: seed reverse directed friendship row")

	s.deps.SeedFriendship(otherUserID, currentUserID, status)

	pending, accepted, blocked := s.deps.FriendshipCounts()
	fmt.Printf("Output: pending=%d accepted=%d blocked=%d\n", pending, accepted, blocked)
	fmt.Println("Mutation: reverse friendship row added")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iSearchFriendsByName(name string) error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: an authenticated user searches the friendship directory")
	fmt.Printf("Input: name=%s auth_header_present=%t\n", name, authHeader != "")
	fmt.Println("Action: GET /api/user/search")

	path := fmt.Sprintf("/api/user/search?name=%s", name)
	if err := s.DoRequestWithHeaders(http.MethodGet, path, s.authHeaders(authHeader)); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) searchResultsShouldIncludeWithPublicIDAvatarAndFriendshipStatus(name, publicID, avatar, friendshipStatus string) error {
	start := time.Now()
	fmt.Println("Given: search results should include a matched user with friendship status")
	fmt.Printf("Input: name=%s public_id=%s avatar=%s status=%s\n", name, publicID, avatar, friendshipStatus)
	fmt.Println("Action: decode search response")

	results, err := decodeSearchResults(s.ResponseBody)
	if err != nil {
		return err
	}
	for _, item := range results {
		if item.Name == name && item.PublicID == publicID && item.Avatar == avatar && item.FriendshipStatus != nil && *item.FriendshipStatus == friendshipStatus {
			fmt.Printf("Output: matched_user_id=%d matched=true\n", item.UserID)
			fmt.Println("Mutation: none")
			fmt.Printf("Duration: %s\n", time.Since(start))
			return nil
		}
	}
	return fmt.Errorf("expected search results to include %s with status %s; body=%s", name, friendshipStatus, string(s.ResponseBody))
}

func (s *steps) searchResultsShouldIncludeWithPublicIDAvatarAndNoFriendshipStatus(name, publicID, avatar string) error {
	start := time.Now()
	fmt.Println("Given: search results should include a matched user with no friendship status")
	fmt.Printf("Input: name=%s public_id=%s avatar=%s\n", name, publicID, avatar)
	fmt.Println("Action: decode search response")

	results, err := decodeSearchResults(s.ResponseBody)
	if err != nil {
		return err
	}
	for _, item := range results {
		if item.Name == name && item.PublicID == publicID && item.Avatar == avatar && item.FriendshipStatus == nil {
			fmt.Printf("Output: matched_user_id=%d matched=true\n", item.UserID)
			fmt.Println("Mutation: none")
			fmt.Printf("Duration: %s\n", time.Since(start))
			return nil
		}
	}
	return fmt.Errorf("expected search results to include %s without friendship status; body=%s", name, string(s.ResponseBody))
}

func (s *steps) searchResultsShouldNotIncludeTheLoggedInUser() error {
	start := time.Now()
	fmt.Println("Given: search results should hide the requesting user")
	fmt.Printf("Input: current_user_id=%d\n", s.accountBDD.LastSessionUserID())
	fmt.Println("Action: decode search response")

	results, err := decodeSearchResults(s.ResponseBody)
	if err != nil {
		return err
	}
	currentUserID := int64(s.accountBDD.LastSessionUserID())
	for _, item := range results {
		if item.UserID == currentUserID {
			return fmt.Errorf("expected current user %d to be filtered out", currentUserID)
		}
	}
	fmt.Println("Output: current_user_hidden=true")
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (s *steps) iApplyToUser(userID int64) error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: an authenticated user applies for friendship")
	fmt.Printf("Input: friend_id=%d auth_header_present=%t\n", userID, authHeader != "")
	fmt.Println("Action: POST /api/user/friends/apply")

	payload := map[string]any{"friend_id": userID}
	if err := s.doJSONRequestWithHeaders(http.MethodPost, "/api/user/friends/apply", payload, s.authHeaders(authHeader)); err != nil {
		return err
	}

	pending, accepted, blocked := s.deps.FriendshipCounts()
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Printf("Mutation: pending=%d accepted=%d blocked=%d\n", pending, accepted, blocked)
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iApplyWithAnInvalidFriendPayload() error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: an authenticated user sends an invalid friendship payload")
	fmt.Printf("Input: auth_header_present=%t missing_friend_id=true\n", authHeader != "")
	fmt.Println("Action: POST /api/user/friends/apply")

	if err := s.doJSONRequestWithHeaders(http.MethodPost, "/api/user/friends/apply", map[string]any{}, s.authHeaders(authHeader)); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iApplyToTheLoggedInUser() error {
	return s.iApplyToUser(int64(s.accountBDD.LastSessionUserID()))
}

func (s *steps) friendshipRowsShouldContainFromTheLoggedInUserToUser(status string, userID int64) error {
	start := time.Now()
	currentUserID := s.accountBDD.LastSessionUserID()
	fmt.Println("Given: a friendship mutation should persist a directed row")
	fmt.Printf("Input: from_user_id=%d to_user_id=%d expected_status=%s\n", currentUserID, userID, status)
	fmt.Println("Action: inspect in-memory friendship repository")

	row, ok := s.deps.FindFriendship(currentUserID, shared.UserID(userID))
	if !ok {
		return fmt.Errorf("expected friendship row from %d to %d", currentUserID, userID)
	}
	fmt.Printf("Output: friendship_id=%d actual_status=%s match=%t\n", row.ID, row.Status, string(row.Status) == status)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if string(row.Status) != status {
		return fmt.Errorf("expected status %s, got %s", status, row.Status)
	}
	return nil
}

func (s *steps) iRequestMyFriends() error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: the logged in user requests the friends tab data")
	fmt.Printf("Input: auth_header_present=%t\n", authHeader != "")
	fmt.Println("Action: GET /api/user/friends")

	if err := s.DoRequestWithHeaders(http.MethodGet, "/api/user/friends", s.authHeaders(authHeader)); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) theFriendsResponseShouldIncludeWithStatus(name, status string) error {
	start := time.Now()
	fmt.Println("Given: friends tab data should include a named friendship row")
	fmt.Printf("Input: name=%s expected_status=%s\n", name, status)
	fmt.Println("Action: decode friends response")

	items, err := decodeFriendsResponse(s.ResponseBody)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Name == name && item.Status == status {
			fmt.Printf("Output: friendship_id=%d matched=true\n", item.FriendshipID)
			fmt.Println("Mutation: none")
			fmt.Printf("Duration: %s\n", time.Since(start))
			return nil
		}
	}
	return fmt.Errorf("expected friends response to include %s with status %s; body=%s", name, status, string(s.ResponseBody))
}

func (s *steps) theFriendsResponseShouldNotIncludeUser(userID int64) error {
	start := time.Now()
	fmt.Println("Given: friends tab data should exclude a user")
	fmt.Printf("Input: user_id=%d\n", userID)
	fmt.Println("Action: decode friends response")

	items, err := decodeFriendsResponse(s.ResponseBody)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.UserID == userID {
			return fmt.Errorf("expected friends response not to include user %d", userID)
		}
	}
	fmt.Println("Output: user_hidden=true")
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (s *steps) iRequestSentFriendRequests() error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: the logged in user requests sent pending friendships")
	fmt.Printf("Input: auth_header_present=%t\n", authHeader != "")
	fmt.Println("Action: GET /api/user/friends/sent")

	if err := s.DoRequestWithHeaders(http.MethodGet, "/api/user/friends/sent", s.authHeaders(authHeader)); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) theSentRequestsResponseShouldInclude(name string) error {
	start := time.Now()
	fmt.Println("Given: sent requests data should include an outbound request")
	fmt.Printf("Input: name=%s\n", name)
	fmt.Println("Action: decode sent requests response")

	items, err := decodeRequestResponse(s.ResponseBody)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Name == name {
			fmt.Printf("Output: friendship_id=%d matched=true\n", item.FriendshipID)
			fmt.Println("Mutation: none")
			fmt.Printf("Duration: %s\n", time.Since(start))
			return nil
		}
	}
	return fmt.Errorf("expected sent requests response to include %s; body=%s", name, string(s.ResponseBody))
}

func (s *steps) theSentRequestsResponseShouldNotIncludeUser(userID int64) error {
	start := time.Now()
	fmt.Println("Given: sent requests data should exclude a user")
	fmt.Printf("Input: user_id=%d\n", userID)
	fmt.Println("Action: decode sent requests response")

	items, err := decodeRequestResponse(s.ResponseBody)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.UserID == userID {
			return fmt.Errorf("expected sent requests response not to include user %d", userID)
		}
	}
	fmt.Println("Output: user_hidden=true")
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (s *steps) iRequestIncomingFriendRequests() error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: the logged in user requests inbound pending friendships")
	fmt.Printf("Input: auth_header_present=%t\n", authHeader != "")
	fmt.Println("Action: GET /api/user/friends/requests")

	if err := s.DoRequestWithHeaders(http.MethodGet, "/api/user/friends/requests", s.authHeaders(authHeader)); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) theIncomingRequestsResponseShouldInclude(name string) error {
	start := time.Now()
	fmt.Println("Given: requests tab data should include an inbound request")
	fmt.Printf("Input: name=%s\n", name)
	fmt.Println("Action: decode requests response")

	items, err := decodeRequestResponse(s.ResponseBody)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Name == name {
			fmt.Printf("Output: friendship_id=%d matched=true\n", item.FriendshipID)
			fmt.Println("Mutation: none")
			fmt.Printf("Duration: %s\n", time.Since(start))
			return nil
		}
	}
	return fmt.Errorf("expected requests response to include %s; body=%s", name, string(s.ResponseBody))
}

func (s *steps) theIncomingRequestsResponseShouldNotIncludeUser(userID int64) error {
	start := time.Now()
	fmt.Println("Given: requests tab data should exclude a specific user")
	fmt.Printf("Input: user_id=%d\n", userID)
	fmt.Println("Action: decode requests response")

	items, err := decodeRequestResponse(s.ResponseBody)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.UserID == userID {
			return fmt.Errorf("expected requests response not to include user %d", userID)
		}
	}
	fmt.Println("Output: user_hidden=true")
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (s *steps) iRequestBlockedUsers() error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: the logged in user requests blocked friendships")
	fmt.Printf("Input: auth_header_present=%t\n", authHeader != "")
	fmt.Println("Action: GET /api/user/block")

	if err := s.DoRequestWithHeaders(http.MethodGet, "/api/user/block", s.authHeaders(authHeader)); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) theBlockedResponseShouldIncludeWithStatus(name, status string) error {
	start := time.Now()
	fmt.Println("Given: blocked users data should include a named relationship row")
	fmt.Printf("Input: name=%s expected_status=%s\n", name, status)
	fmt.Println("Action: decode blocked response")

	items, err := decodeFriendsResponse(s.ResponseBody)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Name == name && item.Status == status {
			fmt.Printf("Output: friendship_id=%d matched=true\n", item.FriendshipID)
			fmt.Println("Mutation: none")
			fmt.Printf("Duration: %s\n", time.Since(start))
			return nil
		}
	}
	return fmt.Errorf("expected blocked response to include %s with status %s; body=%s", name, status, string(s.ResponseBody))
}

func (s *steps) theBlockedResponseShouldNotIncludeUser(userID int64) error {
	start := time.Now()
	fmt.Println("Given: blocked users data should exclude a user")
	fmt.Printf("Input: user_id=%d\n", userID)
	fmt.Println("Action: decode blocked response")

	items, err := decodeFriendsResponse(s.ResponseBody)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.UserID == userID {
			return fmt.Errorf("expected blocked response not to include user %d", userID)
		}
	}
	fmt.Println("Output: user_hidden=true")
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (s *steps) iAcceptTheInboundFriendRequestFromUser(userID int64) error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	currentUserID := s.accountBDD.LastSessionUserID()
	row, ok := s.deps.FindFriendship(shared.UserID(userID), currentUserID)
	if !ok {
		return fmt.Errorf("no inbound friendship found from user %d", userID)
	}
	fmt.Println("Given: an inbound pending request exists and can be accepted")
	fmt.Printf("Input: friendship_id=%d from_user_id=%d to_user_id=%d\n", row.ID, userID, currentUserID)
	fmt.Println("Action: POST /api/user/friends/:id/accept")

	path := fmt.Sprintf("/api/user/friends/%d/accept", row.ID)
	if err := s.doJSONRequestWithHeaders(http.MethodPost, path, nil, s.authHeaders(authHeader)); err != nil {
		return err
	}

	pending, accepted, blocked := s.deps.FriendshipCounts()
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Printf("Mutation: pending=%d accepted=%d blocked=%d\n", pending, accepted, blocked)
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iAcceptFriendshipID(friendshipID int64) error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: an authenticated user accepts a friendship by id")
	fmt.Printf("Input: friendship_id=%d auth_header_present=%t\n", friendshipID, authHeader != "")
	fmt.Println("Action: POST /api/user/friends/:id/accept")

	path := fmt.Sprintf("/api/user/friends/%d/accept", friendshipID)
	if err := s.doJSONRequestWithHeaders(http.MethodPost, path, nil, s.authHeaders(authHeader)); err != nil {
		return err
	}

	pending, accepted, blocked := s.deps.FriendshipCounts()
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Printf("Mutation: pending=%d accepted=%d blocked=%d\n", pending, accepted, blocked)
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iRejectTheInboundFriendRequestFromUser(userID int64) error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	currentUserID := s.accountBDD.LastSessionUserID()
	row, ok := s.deps.FindFriendship(shared.UserID(userID), currentUserID)
	if !ok {
		return fmt.Errorf("no inbound friendship found from user %d", userID)
	}
	fmt.Println("Given: an inbound pending request exists and can be rejected")
	fmt.Printf("Input: friendship_id=%d from_user_id=%d to_user_id=%d\n", row.ID, userID, currentUserID)
	fmt.Println("Action: DELETE /api/user/friends/:id")

	path := fmt.Sprintf("/api/user/friends/%d", row.ID)
	if err := s.DoRequestWithHeaders(http.MethodDelete, path, s.authHeaders(authHeader)); err != nil {
		return err
	}

	pending, accepted, blocked := s.deps.FriendshipCounts()
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Printf("Mutation: pending=%d accepted=%d blocked=%d\n", pending, accepted, blocked)
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iCancelTheSentFriendRequestToUser(userID int64) error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	currentUserID := s.accountBDD.LastSessionUserID()
	row, ok := s.deps.FindFriendship(currentUserID, shared.UserID(userID))
	if !ok {
		return fmt.Errorf("no outbound friendship found to user %d", userID)
	}
	fmt.Println("Given: an outbound pending request exists and can be cancelled")
	fmt.Printf("Input: friendship_id=%d from_user_id=%d to_user_id=%d\n", row.ID, currentUserID, userID)
	fmt.Println("Action: DELETE /api/user/friends/sent/:id")

	path := fmt.Sprintf("/api/user/friends/sent/%d", row.ID)
	if err := s.DoRequestWithHeaders(http.MethodDelete, path, s.authHeaders(authHeader)); err != nil {
		return err
	}

	pending, accepted, blocked := s.deps.FriendshipCounts()
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Printf("Mutation: pending=%d accepted=%d blocked=%d\n", pending, accepted, blocked)
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iUnfriendUser(userID int64) error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	currentUserID := s.accountBDD.LastSessionUserID()
	row, ok := s.deps.FindFriendship(currentUserID, shared.UserID(userID))
	if !ok {
		return fmt.Errorf("no friendship found to user %d", userID)
	}
	fmt.Println("Given: an accepted friendship exists and can be removed")
	fmt.Printf("Input: friendship_id=%d from_user_id=%d to_user_id=%d\n", row.ID, currentUserID, userID)
	fmt.Println("Action: DELETE /api/user/friends/:id")

	path := fmt.Sprintf("/api/user/friends/%d", row.ID)
	if err := s.DoRequestWithHeaders(http.MethodDelete, path, s.authHeaders(authHeader)); err != nil {
		return err
	}

	pending, accepted, blocked := s.deps.FriendshipCounts()
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Printf("Mutation: pending=%d accepted=%d blocked=%d\n", pending, accepted, blocked)
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iBlockUser(userID int64) error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: an authenticated user blocks another user")
	fmt.Printf("Input: target_user_id=%d auth_header_present=%t\n", userID, authHeader != "")
	fmt.Println("Action: POST /api/user/block/:id")

	path := fmt.Sprintf("/api/user/block/%d", userID)
	if err := s.doJSONRequestWithHeaders(http.MethodPost, path, nil, s.authHeaders(authHeader)); err != nil {
		return err
	}

	pending, accepted, blocked := s.deps.FriendshipCounts()
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Printf("Mutation: pending=%d accepted=%d blocked=%d\n", pending, accepted, blocked)
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) iUnblockUser(userID int64) error {
	s.start = time.Now()
	authHeader, err := s.authorizationHeader()
	if err != nil {
		return err
	}
	fmt.Println("Given: an authenticated user unblocks another user")
	fmt.Printf("Input: target_user_id=%d auth_header_present=%t\n", userID, authHeader != "")
	fmt.Println("Action: DELETE /api/user/block/:id")

	path := fmt.Sprintf("/api/user/block/%d", userID)
	if err := s.DoRequestWithHeaders(http.MethodDelete, path, s.authHeaders(authHeader)); err != nil {
		return err
	}

	pending, accepted, blocked := s.deps.FriendshipCounts()
	fmt.Printf("Output: status=%d body=%s\n", s.Response.StatusCode, string(s.ResponseBody))
	fmt.Printf("Mutation: pending=%d accepted=%d blocked=%d\n", pending, accepted, blocked)
	fmt.Printf("Duration: %s\n", time.Since(s.start))
	return nil
}

func (s *steps) mutualFriendshipRowsBetweenTheLoggedInUserAndUserShouldBothBe(userID int64, status string) error {
	start := time.Now()
	currentUserID := s.accountBDD.LastSessionUserID()
	fmt.Println("Given: accepting a request should create mutual directed friendships")
	fmt.Printf("Input: current_user_id=%d other_user_id=%d expected_status=%s\n", currentUserID, userID, status)
	fmt.Println("Action: inspect both friendship directions")

	forward, ok := s.deps.FindFriendship(currentUserID, shared.UserID(userID))
	if !ok {
		return fmt.Errorf("expected forward friendship row from %d to %d", currentUserID, userID)
	}
	reverse, ok := s.deps.FindFriendship(shared.UserID(userID), currentUserID)
	if !ok {
		return fmt.Errorf("expected reverse friendship row from %d to %d", userID, currentUserID)
	}
	match := strings.EqualFold(string(forward.Status), status) && strings.EqualFold(string(reverse.Status), status)
	fmt.Printf("Output: forward_status=%s reverse_status=%s match=%t\n", forward.Status, reverse.Status, match)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	if !match {
		return fmt.Errorf("expected both rows to be %s, got forward=%s reverse=%s", status, forward.Status, reverse.Status)
	}
	return nil
}

func (s *steps) friendshipRowsBetweenTheLoggedInUserAndUserShouldNotExist(userID int64) error {
	start := time.Now()
	currentUserID := s.accountBDD.LastSessionUserID()
	fmt.Println("Given: a friendship mutation should remove all rows between two users")
	fmt.Printf("Input: current_user_id=%d other_user_id=%d\n", currentUserID, userID)
	fmt.Println("Action: inspect both friendship directions")

	if _, ok := s.deps.FindFriendship(currentUserID, shared.UserID(userID)); ok {
		return fmt.Errorf("expected no friendship row from %d to %d", currentUserID, userID)
	}
	if _, ok := s.deps.FindFriendship(shared.UserID(userID), currentUserID); ok {
		return fmt.Errorf("expected no friendship row from %d to %d", userID, currentUserID)
	}
	fmt.Println("Output: friendship_absent=true")
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (s *steps) friendshipRowFromUserToTheLoggedInUserShouldNotExist(userID int64) error {
	start := time.Now()
	currentUserID := s.accountBDD.LastSessionUserID()
	fmt.Println("Given: a friendship mutation should remove a reverse directed row")
	fmt.Printf("Input: from_user_id=%d to_user_id=%d\n", userID, currentUserID)
	fmt.Println("Action: inspect reverse friendship direction")

	if _, ok := s.deps.FindFriendship(shared.UserID(userID), currentUserID); ok {
		return fmt.Errorf("expected no friendship row from %d to %d", userID, currentUserID)
	}
	fmt.Println("Output: reverse_row_absent=true")
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (s *steps) authorizationHeader() (string, error) {
	if s.authHeader != "" {
		return s.authHeader, nil
	}
	if s.Response == nil {
		return "", fmt.Errorf("no login response available")
	}
	header := s.Response.Header.Get("Authorization")
	if header == "" {
		return "", fmt.Errorf("no Authorization header from login response")
	}
	s.authHeader = header
	return header, nil
}

func (s *steps) authHeaders(authHeader string) map[string]string {
	deviceID := s.accountBDD.LastSessionDeviceID().String()
	return map[string]string{
		"Authorization": authHeader,
		"X-Device-ID":   deviceID,
	}
}

func (s *steps) doJSONRequestWithHeaders(method, path string, payload any, headers map[string]string) error {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return err
		}
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

func decodeSearchResults(body []byte) ([]userSearchResponse, error) {
	var items []userSearchResponse
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func decodeFriendsResponse(body []byte) ([]friendResponse, error) {
	var items []friendResponse
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func decodeRequestResponse(body []byte) ([]requestResponse, error) {
	var items []requestResponse
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *steps) aDirectChatRoomShouldExistBetweenLoggedInUserAndUser(userID int64) error {
	start := time.Now()
	currentUserID := s.accountBDD.LastSessionUserID()
	fmt.Println("Given: accepting a friend request should auto-create a direct chat room")
	fmt.Printf("Input: current_user_id=%d friend_user_id=%d\n", currentUserID, userID)
	fmt.Println("Action: inspect in-memory chat room repository")

	room, ok := s.deps.FindDirectRoomBetweenUsers(currentUserID, shared.UserID(userID))
	if !ok {
		return fmt.Errorf("expected a direct chat room between user %d and user %d, but none was found", currentUserID, userID)
	}
	s.lastDirectRoomID = room.ID
	fmt.Printf("Output: room_id=%d room_type=%s found=true\n", room.ID, room.Type)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

func (s *steps) bothMembersShouldHaveRoleInThatDirectRoom(expectedRole string) error {
	start := time.Now()
	if s.lastDirectRoomID == 0 {
		return fmt.Errorf("no direct room found in previous step; run 'a direct chat room should exist' first")
	}
	fmt.Println("Given: both members of the direct room should have the expected role")
	fmt.Printf("Input: room_id=%d expected_role=%s\n", s.lastDirectRoomID, expectedRole)
	fmt.Println("Action: inspect in-memory chat member repository")

	members := s.deps.FindMembersByRoom(s.lastDirectRoomID)
	if len(members) != 2 {
		return fmt.Errorf("expected 2 members in direct room %d, got %d", s.lastDirectRoomID, len(members))
	}
	for _, m := range members {
		if string(m.Role) != expectedRole {
			return fmt.Errorf("expected member %d to have role %s, got %s", m.ID, expectedRole, m.Role)
		}
	}
	fmt.Printf("Output: member_count=%d all_have_role=%s\n", len(members), chatmember.Role(expectedRole))
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))
	return nil
}

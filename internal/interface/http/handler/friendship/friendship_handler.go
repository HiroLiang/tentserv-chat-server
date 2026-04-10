package friendship

import (
	"net/http"
	"strconv"

	friendshipUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/friendship/usecase"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/adapter"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/response"
	"github.com/gin-gonic/gin"
)

type FriendshipHandler struct {
	getFriendsUseCase        *friendshipUseCase.GetFriendsUseCase
	getBlockedUsersUseCase   *friendshipUseCase.GetBlockedUsersUseCase
	applyFriendshipUseCase   *friendshipUseCase.ApplyFriendshipUseCase
	acceptFriendshipUseCase  *friendshipUseCase.AcceptFriendshipUseCase
	getFriendRequestsUseCase *friendshipUseCase.GetFriendRequestsUseCase
	removeFriendshipUseCase  *friendshipUseCase.RemoveFriendshipUseCase
	getSentRequestsUseCase   *friendshipUseCase.GetSentRequestsUseCase
	cancelSentRequestUseCase *friendshipUseCase.CancelSentRequestUseCase
	blockUserUseCase         *friendshipUseCase.BlockUserUseCase
	unblockUserUseCase       *friendshipUseCase.UnblockUserUseCase
}

func NewFriendshipHandler(
	getFriendsUseCase *friendshipUseCase.GetFriendsUseCase,
	getBlockedUsersUseCase *friendshipUseCase.GetBlockedUsersUseCase,
	applyFriendshipUseCase *friendshipUseCase.ApplyFriendshipUseCase,
	acceptFriendshipUseCase *friendshipUseCase.AcceptFriendshipUseCase,
	getFriendRequestsUseCase *friendshipUseCase.GetFriendRequestsUseCase,
	removeFriendshipUseCase *friendshipUseCase.RemoveFriendshipUseCase,
	getSentRequestsUseCase *friendshipUseCase.GetSentRequestsUseCase,
	cancelSentRequestUseCase *friendshipUseCase.CancelSentRequestUseCase,
	blockUserUseCase *friendshipUseCase.BlockUserUseCase,
	unblockUserUseCase *friendshipUseCase.UnblockUserUseCase,
) *FriendshipHandler {
	return &FriendshipHandler{
		getFriendsUseCase:        getFriendsUseCase,
		getBlockedUsersUseCase:   getBlockedUsersUseCase,
		applyFriendshipUseCase:   applyFriendshipUseCase,
		acceptFriendshipUseCase:  acceptFriendshipUseCase,
		getFriendRequestsUseCase: getFriendRequestsUseCase,
		removeFriendshipUseCase:  removeFriendshipUseCase,
		getSentRequestsUseCase:   getSentRequestsUseCase,
		cancelSentRequestUseCase: cancelSentRequestUseCase,
		blockUserUseCase:         blockUserUseCase,
		unblockUserUseCase:       unblockUserUseCase,
	}
}

// @Summary Get blocked users list
// @Description Get the current user's blocked users with name and avatar
// @Tags User
// @Produce json
// @Security BearerAuth
// @Success 200 {array} FriendResponse
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/block [get]
func (h *FriendshipHandler) getBlockedUsers(c *gin.Context) {
	input := adapter.BuildEmptyInput(c)
	out, err := h.getBlockedUsersUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleFriendshipError(c, err)
		return
	}

	resp := make([]FriendResponse, 0, len(out.Blocked))
	for _, f := range out.Blocked {
		resp = append(resp, FriendResponse{
			FriendshipID: f.FriendshipID,
			UserID:       f.UserID,
			Name:         f.Name,
			Avatar:       f.Avatar,
			Status:       f.Status,
			BlockedBy:    f.BlockedBy,
			CreatedAt:    f.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Get friends list
// @Description Get the current user's accepted friends with name and avatar
// @Tags User
// @Produce json
// @Security BearerAuth
// @Success 200 {array} FriendResponse
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/friends [get]
func (h *FriendshipHandler) getFriends(c *gin.Context) {
	input := adapter.BuildEmptyInput(c)
	out, err := h.getFriendsUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleFriendshipError(c, err)
		return
	}

	resp := make([]FriendResponse, 0, len(out.Friends))
	for _, f := range out.Friends {
		resp = append(resp, FriendResponse{
			FriendshipID: f.FriendshipID,
			UserID:       f.UserID,
			Name:         f.Name,
			Avatar:       f.Avatar,
			Status:       f.Status,
			BlockedBy:    f.BlockedBy,
			CreatedAt:    f.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Apply for friendship
// @Description Send a friend request to another user
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ApplyFriendRequest true "Friend request payload"
// @Success 200 {object} nil
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 409 {object} response.ErrorResponse "Friendship already exists"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/friends/apply [post]
func (h *FriendshipHandler) applyFriend(c *gin.Context) {
	var req ApplyFriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": response.ErrorResponse{Code: "INVALID_REQUEST", Message: "invalid friend request payload"}})
		return
	}
	input := adapter.BuildInput(c, friendshipUseCase.ApplyFriendshipInput{FriendID: req.FriendID})
	if err := h.applyFriendshipUseCase.Execute(c.Request.Context(), input); err != nil {
		HandleFriendshipError(c, err)
		return
	}
	c.JSON(http.StatusOK, nil)
}

// @Summary Accept a friend request
// @Description Accept a pending friend request by friendship ID
// @Tags User
// @Produce json
// @Security BearerAuth
// @Param id path int true "Friendship ID"
// @Success 200 {object} nil
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Not Found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/friends/{id}/accept [post]
func (h *FriendshipHandler) acceptFriend(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": response.ErrorResponse{Code: "INVALID_ID", Message: "invalid friendship id"}})
		return
	}
	input := adapter.BuildInput(c, friendshipUseCase.AcceptFriendshipInput{FriendshipID: id})
	if err := h.acceptFriendshipUseCase.Execute(c.Request.Context(), input); err != nil {
		HandleFriendshipError(c, err)
		return
	}
	c.JSON(http.StatusOK, nil)
}

// @Summary Get incoming friend requests
// @Description List pending friend requests where the current user is the recipient
// @Tags User
// @Produce json
// @Security BearerAuth
// @Success 200 {array} FriendRequestResponse
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/friends/requests [get]
func (h *FriendshipHandler) getFriendRequests(c *gin.Context) {
	input := adapter.BuildEmptyInput(c)
	out, err := h.getFriendRequestsUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleFriendshipError(c, err)
		return
	}

	resp := make([]FriendRequestResponse, 0, len(out.Requests))
	for _, r := range out.Requests {
		resp = append(resp, FriendRequestResponse{
			FriendshipID: r.FriendshipID,
			UserID:       r.UserID,
			Name:         r.Name,
			Avatar:       r.Avatar,
			CreatedAt:    r.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Remove a friendship or cancel/reject a friend request
// @Description Delete a friendship record by ID. The caller must be either the initiator or the recipient.
// @Tags User
// @Produce json
// @Security BearerAuth
// @Param id path int true "Friendship ID"
// @Success 200 {object} nil
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Not Found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/friends/{id} [delete]
func (h *FriendshipHandler) removeFriend(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": response.ErrorResponse{Code: "INVALID_ID", Message: "invalid friendship id"}})
		return
	}
	input := adapter.BuildInput(c, friendshipUseCase.RemoveFriendshipInput{FriendshipID: id})
	out, err := h.removeFriendshipUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleFriendshipError(c, err)
		return
	}

	resp := RemoveFriendResponse{}
	if out != nil && out.DeletedDirectRoom != nil {
		resp.DeletedDirectRoom = &RemovedDirectRoomResponse{
			RoomID:    out.DeletedDirectRoom.RoomID,
			MemberIDs: out.DeletedDirectRoom.MemberIDs,
		}
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Get sent friend requests
// @Description List pending friend requests sent by the current user
// @Tags User
// @Produce json
// @Security BearerAuth
// @Success 200 {array} FriendRequestResponse
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/friends/sent [get]
func (h *FriendshipHandler) getSentRequests(c *gin.Context) {
	input := adapter.BuildEmptyInput(c)
	out, err := h.getSentRequestsUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleFriendshipError(c, err)
		return
	}

	resp := make([]FriendRequestResponse, 0, len(out.Requests))
	for _, r := range out.Requests {
		resp = append(resp, FriendRequestResponse{
			FriendshipID: r.FriendshipID,
			UserID:       r.UserID,
			Name:         r.Name,
			Avatar:       r.Avatar,
			CreatedAt:    r.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Cancel a sent friend request
// @Description Cancel a pending friend request that the current user sent
// @Tags User
// @Produce json
// @Security BearerAuth
// @Param id path int true "Friendship ID"
// @Success 200 {object} nil
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Not Found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/friends/sent/{id} [delete]
func (h *FriendshipHandler) cancelSentRequest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": response.ErrorResponse{Code: "INVALID_ID", Message: "invalid friendship id"}})
		return
	}
	input := adapter.BuildInput(c, friendshipUseCase.CancelSentRequestInput{FriendshipID: id})
	if err := h.cancelSentRequestUseCase.Execute(c.Request.Context(), input); err != nil {
		HandleFriendshipError(c, err)
		return
	}
	c.JSON(http.StatusOK, nil)
}

// @Summary Block a user
// @Description Block another user, preventing them from sending friend requests
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Target User ID"
// @Success 200 {object} nil
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 409 {object} response.ErrorResponse "Already blocked"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/block/{id} [post]
func (h *FriendshipHandler) blockUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": response.ErrorResponse{Code: "INVALID_ID", Message: "invalid user id"}})
		return
	}
	input := adapter.BuildInput(c, friendshipUseCase.BlockUserInput{TargetUserID: id})
	if err := h.blockUserUseCase.Execute(c.Request.Context(), input); err != nil {
		HandleFriendshipError(c, err)
		return
	}
	c.JSON(http.StatusOK, nil)
}

// @Summary Unblock a user
// @Description Remove a block on another user
// @Tags User
// @Produce json
// @Security BearerAuth
// @Param id path int true "Target User ID"
// @Success 200 {object} nil
// @Failure 400 {object} response.ErrorResponse "Bad Request or not blocked"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 404 {object} response.ErrorResponse "Not Found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/block/{id} [delete]
func (h *FriendshipHandler) unblockUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": response.ErrorResponse{Code: "INVALID_ID", Message: "invalid user id"}})
		return
	}
	input := adapter.BuildInput(c, friendshipUseCase.UnblockUserInput{TargetUserID: id})
	if err := h.unblockUserUseCase.Execute(c.Request.Context(), input); err != nil {
		HandleFriendshipError(c, err)
		return
	}
	c.JSON(http.StatusOK, nil)
}

func (h *FriendshipHandler) RegisterFriendshipRoutes(r *gin.RouterGroup) {
	r.GET("/friends", h.getFriends)
	r.POST("/friends/apply", h.applyFriend)
	r.POST("/friends/:id/accept", h.acceptFriend)
	r.GET("/friends/requests", h.getFriendRequests)
	r.DELETE("/friends/:id", h.removeFriend)
	r.GET("/friends/sent", h.getSentRequests)
	r.DELETE("/friends/sent/:id", h.cancelSentRequest)
	r.GET("/block", h.getBlockedUsers)
	r.POST("/block/:id", h.blockUser)
	r.DELETE("/block/:id", h.unblockUser)
}

package user

import (
	"net/http"
	"strconv"
	"time"

	friendshipUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/friendship/usecase"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/adapter"
	"github.com/gin-gonic/gin"
)

type FriendResponse struct {
	FriendshipID int64     `json:"friendship_id"`
	UserID       int64     `json:"user_id"`
	Name         string    `json:"name"`
	Avatar       string    `json:"avatar"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type FriendRequestResponse struct {
	FriendshipID int64     `json:"friendship_id"`
	UserID       int64     `json:"user_id"`
	Name         string    `json:"name"`
	Avatar       string    `json:"avatar"`
	CreatedAt    time.Time `json:"created_at"`
}

type ApplyFriendRequest struct {
	FriendID int64 `json:"friend_id" binding:"required"`
}

type FriendshipHandler struct {
	getFriendsUseCase        *friendshipUseCase.GetFriendsUseCase
	applyFriendshipUseCase   *friendshipUseCase.ApplyFriendshipUseCase
	acceptFriendshipUseCase  *friendshipUseCase.AcceptFriendshipUseCase
	getFriendRequestsUseCase *friendshipUseCase.GetFriendRequestsUseCase
	removeFriendshipUseCase  *friendshipUseCase.RemoveFriendshipUseCase
}

func NewFriendshipHandler(
	getFriendsUseCase *friendshipUseCase.GetFriendsUseCase,
	applyFriendshipUseCase *friendshipUseCase.ApplyFriendshipUseCase,
	acceptFriendshipUseCase *friendshipUseCase.AcceptFriendshipUseCase,
	getFriendRequestsUseCase *friendshipUseCase.GetFriendRequestsUseCase,
	removeFriendshipUseCase *friendshipUseCase.RemoveFriendshipUseCase,
) *FriendshipHandler {
	return &FriendshipHandler{
		getFriendsUseCase:        getFriendsUseCase,
		applyFriendshipUseCase:   applyFriendshipUseCase,
		acceptFriendshipUseCase:  acceptFriendshipUseCase,
		getFriendRequestsUseCase: getFriendRequestsUseCase,
		removeFriendshipUseCase:  removeFriendshipUseCase,
	}
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
		_ = c.Error(err)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input := adapter.BuildInput(c, friendshipUseCase.ApplyFriendshipInput{FriendID: req.FriendID})
	if err := h.applyFriendshipUseCase.Execute(c.Request.Context(), input); err != nil {
		_ = c.Error(err)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid friendship id"})
		return
	}
	input := adapter.BuildInput(c, friendshipUseCase.AcceptFriendshipInput{FriendshipID: id})
	if err := h.acceptFriendshipUseCase.Execute(c.Request.Context(), input); err != nil {
		_ = c.Error(err)
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
		_ = c.Error(err)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid friendship id"})
		return
	}
	input := adapter.BuildInput(c, friendshipUseCase.RemoveFriendshipInput{FriendshipID: id})
	if err := h.removeFriendshipUseCase.Execute(c.Request.Context(), input); err != nil {
		_ = c.Error(err)
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
}

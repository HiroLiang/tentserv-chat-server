package user

import (
	"net/http"
	"time"

	friendshipUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/friendship/usecase"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/adapter"
	"github.com/gin-gonic/gin"
)

type FriendResponse struct {
	UserID    int64     `json:"user_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type FriendshipHandler struct {
	getFriendsUseCase *friendshipUseCase.GetFriendsUseCase
}

func NewFriendshipHandler(getFriendsUseCase *friendshipUseCase.GetFriendsUseCase) *FriendshipHandler {
	return &FriendshipHandler{getFriendsUseCase: getFriendsUseCase}
}

// @Summary Get friends list
// @Description Get the current user's accepted friends
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
			UserID:    int64(f.FriendID),
			Status:    string(f.Status),
			CreatedAt: f.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, resp)
}

func (h *FriendshipHandler) RegisterFriendshipRoutes(r *gin.RouterGroup) {
	r.GET("/friends", h.getFriends)
}

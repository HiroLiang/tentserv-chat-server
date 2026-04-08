package friendship

import (
	"errors"
	"net/http"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/response"
	"github.com/gin-gonic/gin"
)

func HandleFriendshipError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, friendship.ErrFriendshipNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": response.ErrNotFound("friendship")})
	case errors.Is(err, friendship.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": response.ErrorResponse{
			Code:    "FORBIDDEN",
			Message: "you are not authorized to perform this action",
		}})
	case errors.Is(err, friendship.ErrFriendshipAlreadyExists), errors.Is(err, friendship.ErrAlreadyFriends):
		c.JSON(http.StatusConflict, gin.H{"error": response.ErrorResponse{
			Code:    "FRIENDSHIP_EXISTS",
			Message: "friendship already exists",
		}})
	case errors.Is(err, friendship.ErrFriendshipNotPending):
		c.JSON(http.StatusBadRequest, gin.H{"error": response.ErrorResponse{
			Code:    "FRIENDSHIP_NOT_PENDING",
			Message: "friendship is not in pending state",
		}})
	case errors.Is(err, friendship.ErrSelfFriendship):
		c.JSON(http.StatusBadRequest, gin.H{"error": response.ErrorResponse{
			Code:    "INVALID_FRIENDSHIP_TARGET",
			Message: "cannot send a friend request to yourself",
		}})
	case errors.Is(err, friendship.ErrAlreadyBlocked):
		c.JSON(http.StatusConflict, gin.H{"error": response.ErrorResponse{
			Code:    "ALREADY_BLOCKED",
			Message: "user is already blocked",
		}})
	case errors.Is(err, friendship.ErrNotBlocked):
		c.JSON(http.StatusBadRequest, gin.H{"error": response.ErrorResponse{
			Code:    "NOT_BLOCKED",
			Message: "user is not blocked",
		}})
	default:
		_ = c.Error(err)
	}
}

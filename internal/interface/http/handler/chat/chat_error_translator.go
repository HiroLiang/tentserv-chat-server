package chat

import (
	"errors"
	"net/http"

	"github.com/HiroLiang/goat-server/internal/domain/chatgroup"
	"github.com/HiroLiang/goat-server/internal/domain/chatmember"
	"github.com/HiroLiang/goat-server/internal/domain/participant"
	"github.com/HiroLiang/goat-server/internal/domain/user"
	"github.com/HiroLiang/goat-server/internal/interface/http/response"
	"github.com/HiroLiang/goat-server/internal/logger"
	"github.com/gin-gonic/gin"
)

func HandleError(c *gin.Context, err error) {
	logger.Log.Error(err.Error())
	switch {
	case errors.Is(err, chatgroup.ErrNotFound):
		c.JSON(http.StatusNotFound, response.ErrNotFound("chat group"))
		return

	case errors.Is(err, chatgroup.ErrDeleted):
		c.JSON(http.StatusGone, response.ErrorResponse{
			Code:    "CHAT_GROUP_DELETED",
			Message: "this chat group no longer exists",
		})
		return

	case errors.Is(err, chatgroup.ErrForbidden):
		c.JSON(http.StatusForbidden, response.ErrInvalid("chat group access"))
		return

	case errors.Is(err, chatgroup.ErrInvalidGroupType):
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    "INVALID_GROUP_TYPE",
			Message: err.Error(),
		})
		return

	case errors.Is(err, chatgroup.ErrFull):
		c.JSON(http.StatusConflict, response.ErrorResponse{
			Code:    "GROUP_FULL",
			Message: "chat group has reached its member limit",
		})
		return

	case errors.Is(err, chatmember.ErrAlreadyMember):
		c.JSON(http.StatusConflict, response.ErrorResponse{
			Code:    "ALREADY_MEMBER",
			Message: "participant is already a member of this group",
		})
		return

	case errors.Is(err, participant.ErrNotFound):
		c.JSON(http.StatusNotFound, response.ErrNotFound("participant"))
		return

	case errors.Is(err, user.ErrInvalidUser):
		c.JSON(http.StatusUnauthorized, response.ErrorResponse{
			Code:    "INVALID_USER",
			Message: "invalid user identity",
		})
		return

	default:
		_ = c.Error(err)
		return
	}
}

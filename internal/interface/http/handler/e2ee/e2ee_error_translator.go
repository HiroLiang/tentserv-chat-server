package e2ee

import (
	"errors"
	"net/http"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/usecase"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/response"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/gin-gonic/gin"
)

func HandleError(c *gin.Context, err error) {
	logger.Log.Error(err.Error())
	switch {
	case errors.Is(err, usecase.ErrInvalidSignature):
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    "INVALID_SIGNATURE",
			Message: "invalid signature or key material",
		})
		return

	case errors.Is(err, usecase.ErrIdentityNotFound):
		c.JSON(http.StatusNotFound, response.ErrorResponse{
			Code:    "IDENTITY_NOT_FOUND",
			Message: "identity key not found",
		})
		return

	case errors.Is(err, usecase.ErrNotRoomMember):
		c.JSON(http.StatusForbidden, response.ErrorResponse{
			Code:    "NOT_ROOM_MEMBER",
			Message: "you are not a member of this room",
		})
		return

	default:
		_ = c.Error(err)
		return
	}
}

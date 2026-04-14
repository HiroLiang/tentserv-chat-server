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
	switch {
	case errors.Is(err, usecase.ErrInvalidSignature):
		logger.Log.Warn(err.Error())
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    "INVALID_SIGNATURE",
			Message: "invalid signature or key material",
		})
		return

	case errors.Is(err, usecase.ErrIdentityNotFound):
		logger.Log.Warn(err.Error())
		c.JSON(http.StatusNotFound, response.ErrorResponse{
			Code:    "IDENTITY_NOT_FOUND",
			Message: "identity key not found",
		})
		return

	case errors.Is(err, usecase.ErrNotRoomMember):
		logger.Log.Warn(err.Error())
		c.JSON(http.StatusForbidden, response.ErrorResponse{
			Code:    "NOT_ROOM_MEMBER",
			Message: "you are not a member of this room",
		})
		return

	case errors.Is(err, usecase.ErrKeyBundleNotFound):
		logger.Log.Warn(err.Error())
		c.JSON(http.StatusNotFound, response.ErrorResponse{
			Code:    "KEY_BUNDLE_NOT_FOUND",
			Message: "key bundle not found",
		})
		return

	case errors.Is(err, usecase.ErrForbidden):
		logger.Log.Warn(err.Error())
		c.JSON(http.StatusForbidden, response.ErrorResponse{
			Code:    "FORBIDDEN",
			Message: "this operation is not permitted",
		})
		return

	case errors.Is(err, usecase.ErrSelfSenderKeySyncIncomplete):
		logger.Log.Warn(err.Error())
		c.JSON(http.StatusConflict, response.ErrorResponse{
			Code:    "SELF_SENDER_KEY_SYNC_INCOMPLETE",
			Message: "self sender key sync is not ready to complete",
		})
		return

	case errors.Is(err, usecase.ErrSelfSenderKeySyncInProgress):
		logger.Log.Warn(err.Error())
		c.JSON(http.StatusConflict, response.ErrorResponse{
			Code:    "SELF_SENDER_KEY_SYNC_IN_PROGRESS",
			Message: "self sender key sync is already in progress",
		})
		return

	default:
		_ = c.Error(err)
		return
	}
}

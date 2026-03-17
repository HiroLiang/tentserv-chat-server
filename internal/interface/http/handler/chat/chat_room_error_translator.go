package chat

import (
	"errors"
	"net/http"

	"github.com/HiroLiang/goat-server/internal/application/chat/usecase"
	"github.com/HiroLiang/goat-server/internal/interface/http/response"
	"github.com/HiroLiang/goat-server/internal/logger"
	"github.com/gin-gonic/gin"
)

func HandleError(c *gin.Context, err error) {
	logger.Log.Error(err.Error())
	switch {
	case errors.Is(err, usecase.ErrChatRoomNotFound):
		c.JSON(http.StatusNotFound, response.ErrNotFound("chat room"))

	case errors.Is(err, usecase.ErrAlreadyMember):
		c.JSON(http.StatusConflict, response.ErrorResponse{
			Code:    "ALREADY_MEMBER",
			Message: "already a member of this room",
		})

	case errors.Is(err, usecase.ErrInvitationAlreadyExists):
		c.JSON(http.StatusConflict, response.ErrorResponse{
			Code:    "INVITATION_EXISTS",
			Message: "join request already pending",
		})

	case errors.Is(err, usecase.ErrInvitationNotFound):
		c.JSON(http.StatusNotFound, response.ErrNotFound("invitation"))

	case errors.Is(err, usecase.ErrInvitationAlreadyResolved):
		c.JSON(http.StatusConflict, response.ErrorResponse{
			Code:    "INVITATION_RESOLVED",
			Message: "invitation already resolved",
		})

	case errors.Is(err, usecase.ErrNotRoomAdmin):
		c.JSON(http.StatusForbidden, response.ErrorResponse{
			Code:    "NOT_ROOM_ADMIN",
			Message: "caller is not room owner or admin",
		})

	case errors.Is(err, usecase.ErrNotRoomMember):
		c.JSON(http.StatusForbidden, response.ErrorResponse{
			Code:    "NOT_ROOM_MEMBER",
			Message: "not a member of this room",
		})

	case errors.Is(err, usecase.ErrParticipantNotFound):
		c.JSON(http.StatusNotFound, response.ErrNotFound("participant"))

	case errors.Is(err, usecase.ErrChatRoomCreate), errors.Is(err, usecase.ErrInvitationCreate):
		c.JSON(http.StatusInternalServerError, response.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "internal server error",
		})

	default:
		_ = c.Error(err)
	}
}

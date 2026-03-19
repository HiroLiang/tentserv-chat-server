package participant

import (
	"errors"
	"net/http"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/chat/usecase"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/response"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/gin-gonic/gin"
)

func HandleError(c *gin.Context, err error) {
	logger.Log.Error(err.Error())
	switch {
	case errors.Is(err, usecase.ErrParticipantAlreadyExists):
		c.JSON(http.StatusConflict, response.ErrorResponse{
			Code:    "PARTICIPANT_EXIST",
			Message: "participant already exists",
		})

	case errors.Is(err, usecase.ErrParticipantNotFound):
		c.JSON(http.StatusNotFound, response.ErrNotFound("participant"))

	case errors.Is(err, usecase.ErrCreateParticipant):
		c.JSON(http.StatusInternalServerError, response.ErrorResponse{
			Code:    "CREATE_PARTICIPANT_FAILED",
			Message: "failed to create participant",
		})

	default:
		_ = c.Error(err)
	}
}

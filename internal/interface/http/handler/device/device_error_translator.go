package device

import (
	"errors"
	"net/http"

	domaindevice "github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/response"
	"github.com/gin-gonic/gin"
)

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domaindevice.ErrDeviceNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domaindevice.ErrDeviceAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domaindevice.ErrInvalidPlatform):
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, user.ErrInvalidUser):
		c.JSON(http.StatusUnauthorized, response.ErrorResponse{
			Code:    "INVALID_USER",
			Message: "invalid user identity",
		})
	default:
		_ = c.Error(err)
	}
}

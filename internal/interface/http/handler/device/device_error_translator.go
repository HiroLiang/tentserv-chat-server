package device

import (
	"errors"
	"net/http"

	deviceUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/device/usecase"
	domaindevice "github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/response"
	"github.com/gin-gonic/gin"
)

func handleError(c *gin.Context, err error) {
	switch {
	// Domain errors
	case errors.Is(err, domaindevice.ErrDeviceNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domaindevice.ErrDeviceAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domaindevice.ErrInvalidPlatform):
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})

	// Application layer errors
	case errors.Is(err, deviceUseCase.ErrDeviceNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, deviceUseCase.ErrDeviceExist):
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, deviceUseCase.ErrInvalidID):
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, deviceUseCase.ErrInvalidPlatform):
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, deviceUseCase.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, response.ErrorResponse{
			Code:    "UNAUTHORIZED",
			Message: "authentication required",
		})
	case errors.Is(err, deviceUseCase.ErrUpdateFailed),
		errors.Is(err, deviceUseCase.ErrDeleteFailed),
		errors.Is(err, deviceUseCase.ErrBindFailed),
		errors.Is(err, deviceUseCase.ErrRegisterFailed):
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})

	default:
		_ = c.Error(err)
	}
}

package device

import (
	"errors"
	"net/http"

	domaindevice "github.com/HiroLiang/goat-server/internal/domain/device"
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
	default:
		_ = c.Error(err)
	}
}

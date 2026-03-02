package device

import (
	"errors"
	"net/http"

	appdevice "github.com/HiroLiang/goat-server/internal/application/device"
	"github.com/HiroLiang/goat-server/internal/domain/device"
	"github.com/HiroLiang/goat-server/internal/interface/http/adapter"
	"github.com/gin-gonic/gin"
)

type DeviceHandler struct {
	deviceUseCase *appdevice.UseCase
}

func NewDeviceHandler(deviceUseCase *appdevice.UseCase) *DeviceHandler {
	return &DeviceHandler{deviceUseCase: deviceUseCase}
}

func (h *DeviceHandler) RegisterDeviceRoutes(r *gin.RouterGroup) {
	r.GET("/:device_id", h.getDeviceInfo)
	r.POST("/register", h.registerDeviceId)
	r.PATCH("/:device_id", h.updateDeviceInfo)
}

// @Summary registerDeviceId
// @Description try to register device id checking is id already registered
// @Tags Device
// @Accept json
// @Produce json
// @Param payload body RegisterDeviceIdRequest true "Register device"
// @Success 201 {object} RegisterDeviceIdResponse
// @Router /api/device/register [post]
func (h *DeviceHandler) registerDeviceId(c *gin.Context) {
	var req RegisterDeviceIdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request: " + err.Error()})
		return
	}

	input := adapter.BuildInput(c, appdevice.RegisterDeviceInput{
		DeviceID: req.DeviceID,
		Name:     req.DeviceName,
		Platform: req.Platform,
	})

	out, err := h.deviceUseCase.RegisterDevice(c.Request.Context(), input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, RegisterDeviceIdResponse{
		Success:    true,
		DeviceID:   out.DeviceID,
		DeviceName: out.Name,
		Platform:   out.Platform,
		CreatedAt:  out.CreatedAt,
	})
}

// @Summary getDeviceInfo
// @Description try to get device info
// @Tags Device
// @Accept json
// @Produce json
// @Param device_id path string true "Device id"
// @Success 200 {object} GetDeviceInfoResponse "Device info"
// @Router /api/device/{device_id} [get]
func (h *DeviceHandler) getDeviceInfo(c *gin.Context) {
	deviceID := c.Param("device_id")

	input := adapter.BuildInput(c, appdevice.GetDeviceInput{
		DeviceID: deviceID,
	})

	out, err := h.deviceUseCase.GetDevice(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			c.JSON(http.StatusOK, GetDeviceInfoResponse{Success: false})
			return
		}
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, GetDeviceInfoResponse{
		Success:    true,
		DeviceID:   out.DeviceID,
		DeviceName: out.Name,
		Platform:   out.Platform,
		CreatedAt:  out.CreatedAt,
	})
}

// @Summary updateDeviceInfo
// @Description try to update device info
// @Tags Device
// @Accept json
// @Produce json
// @Param device_id path string true "Device id"
// @Param payload body DeviceUpdateRequest true "Device update payload"
// @Success 200 {object} DeviceUpdateResponse
// @Router /api/device/{device_id} [patch]
func (h *DeviceHandler) updateDeviceInfo(c *gin.Context) {
	deviceID := c.Param("device_id")

	var req DeviceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request: " + err.Error()})
		return
	}

	input := adapter.BuildInput(c, appdevice.UpdateDeviceInput{
		DeviceID: deviceID,
		Name:     req.DeviceName,
		Platform: req.Platform,
	})

	out, err := h.deviceUseCase.UpdateDevice(c.Request.Context(), input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, DeviceUpdateResponse{
		Success:    true,
		DeviceID:   out.DeviceID,
		DeviceName: out.Name,
		Platform:   out.Platform,
		CreatedAt:  out.CreatedAt,
	})
}

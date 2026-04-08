package device

import (
	"net/http"

	deviceUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/device/usecase"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/adapter"
	"github.com/gin-gonic/gin"
)

type DeviceHandler struct {
	registerUseCase     *deviceUseCase.RegisterUseCase
	getProfileUsecase   *deviceUseCase.GetProfileUseCase
	updateDeviceUseCase *deviceUseCase.UpdateDeviceUseCase
	listDevicesUseCase  *deviceUseCase.ListDevicesUseCase
	bindAccountUseCase  *deviceUseCase.BindAccountUseCase
	deleteDeviceUseCase *deviceUseCase.DeleteDeviceUseCase
}

func NewDeviceHandler(
	registerUseCase *deviceUseCase.RegisterUseCase,
	getDeviceProfileUseCase *deviceUseCase.GetProfileUseCase,
	updateDeviceUseCase *deviceUseCase.UpdateDeviceUseCase,
	listDevicesUseCase *deviceUseCase.ListDevicesUseCase,
	bindAccountUseCase *deviceUseCase.BindAccountUseCase,
	deleteDeviceUseCase *deviceUseCase.DeleteDeviceUseCase,
) *DeviceHandler {
	return &DeviceHandler{
		registerUseCase:     registerUseCase,
		getProfileUsecase:   getDeviceProfileUseCase,
		updateDeviceUseCase: updateDeviceUseCase,
		listDevicesUseCase:  listDevicesUseCase,
		bindAccountUseCase:  bindAccountUseCase,
		deleteDeviceUseCase: deleteDeviceUseCase,
	}
}

// [EN] Device registration is public because it runs before session restoration.
// [中] 裝置註冊在 Session 還原前執行，因此路由不需要驗證。
// [日] 端末登録はセッション復元前に実行されるため、このルートは認証不要にする。
func (h *DeviceHandler) RegisterPublicDeviceRoutes(r *gin.RouterGroup) {
	r.GET("/:device_id", h.getDeviceInfo)
	r.POST("/register", h.registerDeviceId)
	r.PATCH("/:device_id", h.updateDeviceInfo)
}

// RegisterProtectedDeviceRoutes registers routes that require authentication.
func (h *DeviceHandler) RegisterProtectedDeviceRoutes(r *gin.RouterGroup) {
	r.GET("", h.listDevices)
	r.POST("/:device_id/bind", h.bindDevice)
	r.DELETE("/:device_id", h.deleteDevice)
}

// @Summary registerDeviceId
// @Description Register or update a device. Upsert: inserts when missing, updates mutable fields when existing.
// @Tags Device
// @Accept json
// @Produce json
// @Param payload body RegisterDeviceIdRequest true "Register device"
// @Success 200 {object} RegisterDeviceIdResponse
// @Success 201 {object} RegisterDeviceIdResponse
// @Router /api/device/register [post]
func (h *DeviceHandler) registerDeviceId(c *gin.Context) {
	var req RegisterDeviceIdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request: " + err.Error()})
		return
	}

	// [EN] The handler only translates HTTP JSON into usecase input; upsert rules live in the usecase.
	// [中] Handler 只把 HTTP JSON 轉為 usecase input；upsert 規則由 usecase 負責。
	// [日] Handler は HTTP JSON を usecase input に変換するだけで、upsert 規則は usecase が担う。
	input := adapter.BuildInput(c, deviceUseCase.RegisterInput{
		DeviceID: req.DeviceID,
		Name:     req.DeviceName,
		Platform: req.Platform,
	})

	out, err := h.registerUseCase.Execute(c.Request.Context(), input)
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

// @Summary listDevices
// @Description List all devices bound to the authenticated account
// @Tags Device
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ListDevicesResponse
// @Router /api/device [get]
func (h *DeviceHandler) listDevices(c *gin.Context) {
	input := adapter.BuildEmptyInput(c)

	devices, err := h.listDevicesUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		handleError(c, err)
		return
	}

	items := make([]DeviceListItem, 0, len(devices))
	for _, d := range devices {
		items = append(items, DeviceListItem{
			DeviceID:   d.DeviceID,
			DeviceName: d.Name,
			Platform:   d.Platform,
			CreatedAt:  d.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, ListDevicesResponse{
		Success: true,
		Devices: items,
	})
}

// @Summary getDeviceInfo
// @Description Get device info by ID
// @Tags Device
// @Produce json
// @Security BearerAuth
// @Param device_id path string true "Device id"
// @Success 200 {object} GetDeviceInfoResponse
// @Router /api/device/{device_id} [get]
func (h *DeviceHandler) getDeviceInfo(c *gin.Context) {
	deviceID := c.Param("device_id")

	input := adapter.BuildInput(c, deviceUseCase.GetProfileInput{
		DeviceID: deviceID,
	})

	out, err := h.getProfileUsecase.Execute(c.Request.Context(), input)
	if err != nil {
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
// @Description Update device name and platform
// @Tags Device
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param device_id path string true "Device id"
// @Param payload body DeviceUpdateRequest true "Device update payload"
// @Success 200 {object} DeviceUpdateResponse
// @Router /api/device/{device_id} [patch]
func (h *DeviceHandler) updateDeviceInfo(c *gin.Context) {
	var req DeviceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request: " + err.Error()})
		return
	}

	// [EN] The path device_id is authoritative; clients cannot update another device by sending an ID in the body.
	// [中] 以 path device_id 為準，client 不能靠 body 夾帶 ID 去更新另一台裝置。
	// [日] path の device_id を正とし、body に ID を入れて別端末を更新することはできない。
	input := adapter.BuildInput(c, deviceUseCase.UpdateDeviceInput{
		DeviceID: c.Param("device_id"),
		Name:     req.DeviceName,
		Platform: req.Platform,
	})

	out, err := h.updateDeviceUseCase.Execute(c.Request.Context(), input)
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
		UpdatedAt:  out.UpdatedAt,
	})
}

// @Summary bindDevice
// @Description Bind a device to the authenticated account
// @Tags Device
// @Produce json
// @Security BearerAuth
// @Param device_id path string true "Device id"
// @Success 200 {object} BindDeviceResponse
// @Router /api/device/{device_id}/bind [post]
func (h *DeviceHandler) bindDevice(c *gin.Context) {
	input := adapter.BuildInput(c, deviceUseCase.BindAccountInput{
		DeviceID: c.Param("device_id"),
	})

	out, err := h.bindAccountUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, BindDeviceResponse{
		Success:   true,
		DeviceID:  out.DeviceID,
		AccountID: out.AccountID,
	})
}

// @Summary deleteDevice
// @Description Delete a device belonging to the authenticated account
// @Tags Device
// @Produce json
// @Security BearerAuth
// @Param device_id path string true "Device id"
// @Success 204
// @Router /api/device/{device_id} [delete]
func (h *DeviceHandler) deleteDevice(c *gin.Context) {
	input := adapter.BuildInput(c, deviceUseCase.DeleteDeviceInput{
		DeviceID: c.Param("device_id"),
	})

	if err := h.deleteDeviceUseCase.Execute(c.Request.Context(), input); err != nil {
		handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

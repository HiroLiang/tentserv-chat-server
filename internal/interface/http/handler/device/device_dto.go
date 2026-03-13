package device

type RegisterDeviceIdRequest struct {
	DeviceName string `json:"device_name" binding:"required"`
	Platform   string `json:"platform" binding:"required"`
}

type RegisterDeviceIdResponse struct {
	Success    bool   `json:"success"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	Platform   string `json:"platform"`
	CreatedAt  string `json:"created_at"`
}

type GetDeviceInfoResponse struct {
	Success    bool   `json:"success"`
	DeviceID   string `json:"device_id"`
	Platform   string `json:"platform"`
	DeviceName string `json:"device_name"`
	CreatedAt  string `json:"created_at"`
}

type DeviceUpdateRequest struct {
	DeviceName string `json:"device_name" binding:"required"`
	Platform   string `json:"platform" binding:"required"`
}

type DeviceUpdateResponse struct {
	Success    bool   `json:"success"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	Platform   string `json:"platform"`
	CreatedAt  string `json:"created_at"`
}

type BindDeviceRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

type BindDeviceResponse struct {
	Success  bool   `json:"success"`
	DeviceID string `json:"device_id"`
	UserID   int64  `json:"user_id"`
}

package device

type RegisterDeviceInput struct {
	DeviceID string
	Name     string
	Platform string
}

type GetDeviceInput struct {
	DeviceID string
}

type UpdateDeviceInput struct {
	DeviceID string
	Name     string
	Platform string
}

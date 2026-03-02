package device

import "errors"

var (
	ErrDeviceNotFound      = errors.New("device not found")
	ErrDeviceAlreadyExists = errors.New("device already exists")
	ErrInvalidPlatform     = errors.New("unsupported platform, allowed: android, ios, windows, macos, linux, unknown")
)

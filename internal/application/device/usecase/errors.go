package usecase

import "errors"

var (
	ErrUpdateFailed  = errors.New("failed to update device")
	ErrDeleteFailed  = errors.New("failed to delete device")
	ErrBindFailed    = errors.New("failed to bind device")
	ErrUnauthorized  = errors.New("unauthorized")

	ErrRegisterFailed  = errors.New("failed to register device")
	ErrInvalidID       = errors.New("invalid id")
	ErrInvalidPlatform = errors.New("invalid platform")
	ErrDeviceExist     = errors.New("device exist")
	ErrDeviceNotFound  = errors.New("device not found")
)

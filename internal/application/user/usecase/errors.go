package usecase

import "errors"

var (
	ErrUpdateProfile = errors.New("failed to update profile")
	ErrUploadFile    = errors.New("failed to upload file")

	ErrUserNotFound = errors.New("user not found")

	ErrInvalidRoleCode = errors.New("invalid role code")
)

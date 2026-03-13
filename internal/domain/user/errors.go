package user

import "errors"

var (
	ErrInvalidID           = errors.New("invalid id format")
	ErrUserNotFound        = errors.New("user not found")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrInvalidUser         = errors.New("invalid user")
	ErrUserApplying        = errors.New("user registration is pending approval")
	ErrUserBanned          = errors.New("user is banned")
	ErrInvalidPassword     = errors.New("invalid password")
	ErrInvalidEmail        = errors.New("invalid email format")
	ErrGenerateToken       = errors.New("generate token error")
	ErrDefaultRoleNotFound = errors.New("default role not configured")
	ErrInvalidImageType    = errors.New("unsupported image type, allowed: jpeg, png, webp")
	ErrImageTooLarge       = errors.New("image exceeds maximum allowed size")
)

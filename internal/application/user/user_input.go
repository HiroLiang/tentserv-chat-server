package user

import (
	"io"

	"github.com/HiroLiang/goat-server/internal/domain/role"
	"github.com/HiroLiang/goat-server/internal/domain/user"
)

// RegisterInput represents the payload required for user registration.
type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

// LoginInput represents the required fields for a user login request.
// It includes the user's email and password for authentication.
// DeviceID is optional — when provided, the device will be bound to the authenticated user.
type LoginInput struct {
	Email    string
	Password string
	DeviceID string
}

type FindUserRolesInput struct {
	UserID user.ID
}

type AssignRoleInput struct {
	UserID user.ID
	Role   role.Type
}

type RevokeRoleInput struct {
	UserID user.ID
	Role   role.Type
}

// UpdateProfileInput holds the fields a user can update on their profile.
type UpdateProfileInput struct {
	Name string
}

// UploadAvatarInput carries the raw image data for avatar processing.
type UploadAvatarInput struct {
	Image io.Reader
}

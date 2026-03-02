package user

import "github.com/HiroLiang/goat-server/internal/domain/role"

// LoginOutput represents the server's response after a successful login.
// It contains a token and user details.
type LoginOutput struct {
	Token string
}

// CurrentUserOutput let current user logout
type CurrentUserOutput struct {
	ID        int
	Name      string
	Email     string
	AvatarURL string
	CreateAt  string
}

// UploadAvatarOutput contains the new avatar URL after a successful upload.
type UploadAvatarOutput struct {
	AvatarURL string
}

type FindUserRolesOutput struct {
	Roles []role.Type
}

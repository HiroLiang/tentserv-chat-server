package user

// UpdateProfileRequest represents the fields a user can update on their profile.
type UpdateProfileRequest struct {
	Name      string   `json:"name" binding:"required,min=1,max=50"`
	RoleCodes []string `json:"role_codes"`
}

// UpdateProfileResponse is returned after a successful profile update.
type UpdateProfileResponse struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	Avatar    string   `json:"avatar"`
	RoleCodes []string `json:"role_codes"`
}

// UploadAvatarResponse is returned after a successful avatar upload.
type UploadAvatarResponse struct {
	AvatarPath string `json:"avatar_url"`
}

// GetUserProfileResponse is returned for a user profile query.
type GetUserProfileResponse struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	Avatar    string   `json:"avatar"`
	RoleCodes []string `json:"role_codes"`
}

// SwitchUserRequest represents the target user to switch to.
type SwitchUserRequest struct {
	UserID int64 `json:"user_id" binding:"required"`
}

// UserSearchResponse is returned for user search results.
type UserSearchResponse struct {
	UserID           int64   `json:"user_id"`
	Name             string  `json:"name"`
	Avatar           string  `json:"avatar"`
	PublicID         string  `json:"public_id"`
	Account          string  `json:"account"`
	FriendshipStatus *string `json:"friendship_status,omitempty"`
}

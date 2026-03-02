package user

// RegisterRequest represents the payload required for user registration.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// RegisterResponse User register response
type RegisterResponse struct {
	Message string `json:"message"`
}

// LoginRequest represents the required fields for a user login request.
// It includes the user's email and password for authentication.
// DeviceID is optional — when provided, the device will be bound to the authenticated user on success.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	DeviceID string `json:"device_id,omitempty"`
}

// LoginResponse represents the server's response after a successful login.
// It contains a token and user details.
type LoginResponse struct {
	Message string `json:"message"`
}

// CurrentUserResponse queried for user login request.
type CurrentUserResponse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	CreateAt  string `json:"create_at"`
}

// UpdateProfileRequest represents the fields a user can update on their profile.
type UpdateProfileRequest struct {
	Name string `json:"name" binding:"required,min=1,max=50"`
}

// UploadAvatarResponse is returned after a successful avatar upload.
type UploadAvatarResponse struct {
	AvatarURL string `json:"avatar_url"`
}

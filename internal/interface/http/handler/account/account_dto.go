package account

// RegisterRequest represents the payload required for user registration.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Account  string `json:"account" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// RegisterResponse User register response
type RegisterResponse struct {
}

type LoginRequest struct {
	DeviceID   string `json:"device_id" binding:"required"`
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required,min=6"`
}

type LoginResponse struct {
}

type GetProfileResponse struct {
	PublicID    string          `json:"public_id"`
	Email       string          `json:"email"`
	AccountName string          `json:"account_name"`
	Status      string          `json:"status"`
	UserIDs     []int64         `json:"user_ids"`
	CurrentUser UserProfileItem `json:"current_user"`
}

type UserProfileItem struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	Avatar    string   `json:"avatar"`
	RoleCodes []string `json:"role_codes"`
}

type VerifyEmailResponse struct{}

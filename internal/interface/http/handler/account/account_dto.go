package account

// RegisterRequest represents the payload required for user registration.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,max=100"`
	Account  string `json:"account" binding:"required,max=50"`
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=6"`
}

// RegisterResponse User register response
type RegisterResponse struct {
	VerificationToken       string `json:"verification_token"`
	VerificationExpiresAtMS int64  `json:"verification_expires_at_ms"`
}

type LoginRequest struct {
	DeviceID   string `json:"device_id" binding:"required"`
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required,min=6"`
}

type LoginResponse struct {
}

type GetProfileResponse struct {
	AccountID   int64           `json:"account_id"`
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

type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
	Code  string `json:"code" binding:"required,len=6,numeric"`
}

type VerifyEmailResponse struct{}

type ResendVerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type ResendVerifyEmailResponse struct {
	VerificationToken       string `json:"verification_token"`
	VerificationExpiresAtMS int64  `json:"verification_expires_at_ms"`
}

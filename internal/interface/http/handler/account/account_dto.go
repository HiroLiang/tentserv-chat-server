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
	DeviceID string `json:"device_id" binding:"required"`
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginResponse struct {
}

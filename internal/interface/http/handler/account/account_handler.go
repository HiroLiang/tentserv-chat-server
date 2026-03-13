package account

import (
	"net/http"

	authUseCase "github.com/HiroLiang/goat-server/internal/application/auth/usecase"
	"github.com/HiroLiang/goat-server/internal/interface/http/adapter"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	registerUsecase   *authUseCase.RegisterUseCase
	loginUsecase      *authUseCase.LoginUseCase
	logoutUsecase     *authUseCase.LogoutUseCase
	getProfileUsecase *authUseCase.GetProfileUseCase
}

func NewAuthHandler(
	registerUsecase *authUseCase.RegisterUseCase,
	loginUsecase *authUseCase.LoginUseCase,
	logoutUsecase *authUseCase.LogoutUseCase,
	getProfileUsecase *authUseCase.GetProfileUseCase,
) *AuthHandler {
	return &AuthHandler{
		registerUsecase,
		loginUsecase,
		logoutUsecase,
		getProfileUsecase}
}

func (h *AuthHandler) RegisterAuthRoutes(r *gin.RouterGroup) {
	r.POST("/register", h.register)
	r.POST("/login", h.login)
	r.POST("/logout", h.logout)
}

// @Summary Account register
// @Description register with email and password
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body RegisterRequest true "Register messages"
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Not Found"
// @Failure 409 {object} response.ErrorResponse "Conflict"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/auth/register [post]
func (h *AuthHandler) register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, err)
		return
	}

	input := adapter.BuildInput(c, authUseCase.RegisterInput{
		Account:  req.Account,
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
	})

	_, err := h.registerUsecase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{})
}

// @Summary Account Login
// @Description
// Authenticate a user using an email and password.
// If the request already contains an Authorization Bearer token,
// it will be forwarded to the authentication service for session continuity.
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body LoginRequest true "Login payload"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Not Found"
// @Failure 409 {object} response.ErrorResponse "Conflict"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/auth/login [post]
func (h *AuthHandler) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, err)
		return
	}

	input := adapter.BuildInput(c, authUseCase.LoginInput{
		DeviceID:    req.DeviceID,
		AccountName: req.Account,
		Password:    req.Password,
	})

	output, err := h.loginUsecase.Execute(c.Request.Context(), &input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.Header("Authorization", string(output.TokenPair.AccessToken))
	c.JSON(http.StatusOK, LoginResponse{})
}

// @Summary Account Logout
// @Description Remove the session token from the redis store.
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/auth/logout [post]
func (h *AuthHandler) logout(c *gin.Context) {
	input := adapter.BuildInput(c, authUseCase.LogoutInput{})

	_, err := h.logoutUsecase.Execute(c.Request.Context(), &input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

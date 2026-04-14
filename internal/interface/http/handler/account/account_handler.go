package account

import (
	"fmt"
	"net/http"

	authUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/auth/usecase"
	appSecurity "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/security"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/adapter"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/middleware"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/response"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	registerUsecase                      *authUseCase.RegisterUseCase
	loginUsecase                         *authUseCase.LoginUseCase
	logoutUsecase                        *authUseCase.LogoutUseCase
	getProfileUsecase                    *authUseCase.GetProfileUseCase
	verifyEmailUsecase                   *authUseCase.VerifyEmailUseCase
	resendVerifyEmailUsecase             *authUseCase.ResendVerifyEmailUseCase
	verifyLoginDeviceUsecase             *authUseCase.VerifyLoginDeviceUseCase
	resendLoginDeviceVerificationUsecase *authUseCase.ResendLoginDeviceVerificationUseCase
	registerLimiter                      appSecurity.RegisterRateLimiter
}

func NewAuthHandler(
	registerUsecase *authUseCase.RegisterUseCase,
	loginUsecase *authUseCase.LoginUseCase,
	logoutUsecase *authUseCase.LogoutUseCase,
	getProfileUsecase *authUseCase.GetProfileUseCase,
	verifyEmailUsecase *authUseCase.VerifyEmailUseCase,
	resendVerifyEmailUsecase *authUseCase.ResendVerifyEmailUseCase,
	verifyLoginDeviceUsecase *authUseCase.VerifyLoginDeviceUseCase,
	resendLoginDeviceVerificationUsecase *authUseCase.ResendLoginDeviceVerificationUseCase,
	registerLimiter appSecurity.RegisterRateLimiter,
) *AuthHandler {
	return &AuthHandler{
		registerUsecase:                      registerUsecase,
		loginUsecase:                         loginUsecase,
		logoutUsecase:                        logoutUsecase,
		getProfileUsecase:                    getProfileUsecase,
		verifyEmailUsecase:                   verifyEmailUsecase,
		resendVerifyEmailUsecase:             resendVerifyEmailUsecase,
		verifyLoginDeviceUsecase:             verifyLoginDeviceUsecase,
		resendLoginDeviceVerificationUsecase: resendLoginDeviceVerificationUsecase,
		registerLimiter:                      registerLimiter,
	}
}

func (h *AuthHandler) RegisterAuthRoutes(r *gin.RouterGroup) {
	if h.registerLimiter != nil {
		r.POST("/register", middleware.RegisterRateLimitMiddleware(h.registerLimiter), h.register)
	} else {
		r.POST("/register", h.register)
	}
	r.POST("/login", h.login)
	r.POST("/logout", middleware.RequireAuthMiddleware(), h.logout)
	r.GET("/profile", middleware.RequireAuthMiddleware(), h.getProfile)
	r.POST("/verify-email", h.verifyEmail)
	r.POST("/resend-verify-email", h.resendVerifyEmail)
	r.POST("/verify-login-device", h.verifyLoginDevice)
	r.POST("/resend-login-device-verification", h.resendLoginDeviceVerification)
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
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "invalid register payload",
		})
		return
	}

	input := adapter.BuildInput(c, authUseCase.RegisterInput{
		Account:         req.Account,
		Email:           req.Email,
		Name:            req.Name,
		Password:        req.Password,
		ConfirmPassword: req.ConfirmPassword,
	})

	out, err := h.registerUsecase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{
		VerificationToken:       out.VerificationToken,
		VerificationExpiresAtMS: out.VerificationExpiresAtMS,
	})
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
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "invalid login payload",
		})
		return
	}

	input := adapter.BuildInput(c, authUseCase.LoginInput{
		DeviceID:   req.DeviceID,
		Identifier: req.Identifier,
		Password:   req.Password,
	})

	output, err := h.loginUsecase.Execute(c.Request.Context(), &input)
	if err != nil {
		HandleError(c, err)
		return
	}

	if output.LoginStatus == "authenticated" {
		c.Header("Authorization", fmt.Sprintf("Bearer %s", string(output.TokenPair.AccessToken)))
		c.JSON(http.StatusOK, LoginResponse{LoginStatus: output.LoginStatus})
		return
	}

	c.JSON(http.StatusAccepted, LoginResponse{
		LoginStatus:             output.LoginStatus,
		VerificationToken:       output.VerificationToken,
		VerificationExpiresAtMS: output.VerificationExpiresAtMS,
	})
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

// @Summary Get account profile
// @Description Get the current account and user profile
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} GetProfileResponse
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 404 {object} response.ErrorResponse "Not Found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/auth/profile [get]
func (h *AuthHandler) getProfile(c *gin.Context) {
	input := adapter.BuildInput(c, authUseCase.GetProfileInput{})

	out, err := h.getProfileUsecase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, GetProfileResponse{
		AccountID:   out.AccountID,
		PublicID:    out.PublicID,
		Email:       out.Email,
		AccountName: out.AccountName,
		Status:      out.Status,
		UserIDs:     out.UserISs,
		CurrentUser: UserProfileItem{
			ID:        out.CurrentUser.ID,
			Name:      out.CurrentUser.Name,
			Avatar:    out.CurrentUser.Avatar,
			RoleCodes: out.CurrentUser.RoleCodes,
		},
	})
}

// @Summary Verify email address
// @Description Verify account email using the token and 6-digit code sent during registration.
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body VerifyEmailRequest true "Verification payload"
// @Success 200 {object} VerifyEmailResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Router /api/auth/verify-email [post]
func (h *AuthHandler) verifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "invalid verify email payload",
		})
		return
	}

	input := adapter.BuildInput(c, authUseCase.VerifyEmailInput{
		Token: req.Token,
		Code:  req.Code,
	})
	_, err := h.verifyEmailUsecase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, VerifyEmailResponse{})
}

func (h *AuthHandler) verifyLoginDevice(c *gin.Context) {
	var req VerifyLoginDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "invalid verify login device payload",
		})
		return
	}

	input := adapter.BuildInput(c, authUseCase.VerifyLoginDeviceInput{
		Token: req.Token,
		Code:  req.Code,
	})
	out, err := h.verifyLoginDeviceUsecase.Execute(c.Request.Context(), &input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.Header("Authorization", fmt.Sprintf("Bearer %s", string(out.TokenPair.AccessToken)))
	c.JSON(http.StatusOK, VerifyLoginDeviceResponse{LoginStatus: out.LoginStatus})
}

// @Summary Resend verification email
// @Description Resend the verification code for an existing verification session. Generates a new token each call.
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body ResendVerifyEmailRequest true "Token payload"
// @Success 200 {object} ResendVerifyEmailResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/auth/resend-verify-email [post]
func (h *AuthHandler) resendVerifyEmail(c *gin.Context) {
	var req ResendVerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "invalid resend verify email payload",
		})
		return
	}

	input := adapter.BuildInput(c, authUseCase.ResendVerifyEmailInput{Token: req.Token})
	out, err := h.resendVerifyEmailUsecase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, ResendVerifyEmailResponse{
		VerificationToken:       out.VerificationToken,
		VerificationExpiresAtMS: out.VerificationExpiresAtMS,
	})
}

func (h *AuthHandler) resendLoginDeviceVerification(c *gin.Context) {
	var req ResendLoginDeviceVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "invalid resend login device verification payload",
		})
		return
	}

	input := adapter.BuildInput(c, authUseCase.ResendLoginDeviceVerificationInput{Token: req.Token})
	out, err := h.resendLoginDeviceVerificationUsecase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, ResendLoginDeviceVerificationResponse{
		VerificationToken:       out.VerificationToken,
		VerificationExpiresAtMS: out.VerificationExpiresAtMS,
	})
}

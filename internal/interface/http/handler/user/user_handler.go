package user

import (
	"io"
	"mime/multipart"
	"net/http"

	"github.com/HiroLiang/goat-server/internal/application/user"
	domainuser "github.com/HiroLiang/goat-server/internal/domain/user"
	"github.com/HiroLiang/goat-server/internal/interface/http/adapter"
	"github.com/HiroLiang/goat-server/internal/interface/http/middleware"
	"github.com/HiroLiang/goat-server/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserHandler Rest api about user
type UserHandler struct {
	userUseCase *user.UseCase
}

// NewUserHandler Create a new UserHandler instance with dependencies
func NewUserHandler(userUseCase *user.UseCase) *UserHandler {
	return &UserHandler{
		userUseCase: userUseCase,
	}
}

// RegisterUserRoutes registers user-related API routes
func (h *UserHandler) RegisterUserRoutes(r *gin.RouterGroup) {
	r.POST("/register", h.register)
	r.POST("/login", h.login)

	r.POST("/logout", middleware.RequireAuthMiddleware(), h.logout)
	r.GET("/me", middleware.RequireAuthMiddleware(), h.getCurrentUser)
	r.PATCH("/profile", middleware.RequireAuthMiddleware(), h.updateProfile)
	r.POST("/avatar", middleware.RequireAuthMiddleware(), h.uploadAvatar)
}

// @Summary User register
// @Description register with email and password
// @Tags User
// @Accept json
// @Produce json
// @Param payload body RegisterRequest true "Register messages"
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 404 {object} response.ErrorResponse "Not Found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/register [post]
func (h *UserHandler) register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, err)
		return
	}

	data := user.RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}

	if err := h.userUseCase.Register(c.Request.Context(), adapter.BuildInput(c, data)); err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{Message: "Register successful"})
}

// @Summary User Login
// @Description
// Authenticate a user using an email and password.
// If the request already contains an Authorization Bearer token,
// it will be forwarded to the authentication service for session continuity.
// @Tags User
// @Accept json
// @Produce json
// @Param payload body LoginRequest true "Login payload"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 404 {object} response.ErrorResponse "Not Found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/login [post]
func (h *UserHandler) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, err)
		return
	}

	data := user.LoginInput{
		Email:    req.Email,
		Password: req.Password,
		DeviceID: req.DeviceID,
	}

	output, err := h.userUseCase.Login(c.Request.Context(), adapter.BuildInput(c, data))
	if err != nil {
		HandleError(c, err)
		return
	}

	c.Header("Authorization", "Bearer "+output.Token)
	c.JSON(http.StatusOK, LoginResponse{Message: "Login successful"})
}

// @Summary User Logout
// @Description Remove the session token from the redis store.
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/logout [post]
func (h *UserHandler) logout(c *gin.Context) {
	err := h.userUseCase.Logout(c.Request.Context(), adapter.BuildEmptyInput(c))
	if err != nil {
		HandleError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

// @Summary Current User info
// @Description query the current user info
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} CurrentUserResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 404 {object} response.ErrorResponse "Not Found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/me [get]
func (h *UserHandler) getCurrentUser(c *gin.Context) {
	output, err := h.userUseCase.CurrentUserInfo(c.Request.Context(), adapter.BuildEmptyInput(c))
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, CurrentUserResponse{
		ID:        output.ID,
		Name:      output.Name,
		Email:     output.Email,
		AvatarURL: output.AvatarURL,
		CreateAt:  output.CreateAt,
	})
}

// @Summary Update profile
// @Description Update the current user's display name
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body UpdateProfileRequest true "Profile update payload"
// @Success 204
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 404 {object} response.ErrorResponse "Not Found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/profile [patch]
func (h *UserHandler) updateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, err)
		return
	}

	data := user.UpdateProfileInput{Name: req.Name}
	if err := h.userUseCase.UpdateProfile(c.Request.Context(), adapter.BuildInput(c, data)); err != nil {
		HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Upload avatar
// @Description Upload and replace the current user's avatar (jpeg/png/webp, max 5 MB). The image is center-cropped and resized to 256×256.
// @Tags User
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param avatar formData file true "Avatar image file"
// @Success 200 {object} UploadAvatarResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 413 {object} response.ErrorResponse "Image Too Large"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/avatar [post]
func (h *UserHandler) uploadAvatar(c *gin.Context) {
	const maxSize = 5 << 20 // 5 MB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

	file, _, err := c.Request.FormFile("avatar")
	if err != nil {
		if err.Error() == "http: request body too large" {
			HandleError(c, domainuser.ErrImageTooLarge)
			return
		}
		HandleError(c, err)
		return
	}
	defer func(file multipart.File) {
		if err := file.Close(); err != nil {
			logger.Log.Error("failed to close avatar file", zap.Error(err))
		}
	}(file)

	// Detect MIME type from the first 512 bytes, then seek back to start
	buf := make([]byte, 512)
	n, readErr := file.Read(buf)
	if readErr != nil && readErr != io.EOF {
		HandleError(c, domainuser.ErrInvalidImageType)
		return
	}
	if !isAllowedImageType(http.DetectContentType(buf[:n])) {
		HandleError(c, domainuser.ErrInvalidImageType)
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		HandleError(c, err)
		return
	}

	data := user.UploadAvatarInput{Image: file}
	output, err := h.userUseCase.UploadAvatar(c.Request.Context(), adapter.BuildInput(c, data))
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, UploadAvatarResponse{AvatarURL: output.AvatarURL})
}

func isAllowedImageType(mimeType string) bool {
	switch mimeType {
	case "image/jpeg", "image/png", "image/webp":
		return true
	}
	return false
}

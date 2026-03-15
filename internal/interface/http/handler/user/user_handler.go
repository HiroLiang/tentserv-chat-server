package user

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/HiroLiang/goat-server/internal/application/user/usecase"
	"github.com/HiroLiang/goat-server/internal/domain/user"
	"github.com/HiroLiang/goat-server/internal/interface/http/adapter"
	"github.com/HiroLiang/goat-server/internal/interface/http/middleware"
	"github.com/HiroLiang/goat-server/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserHandler Rest api about user
type UserHandler struct {
	updateProfileUseCase *usecase.UpdateProfileUseCase
	uploadAvatarUseCase  *usecase.UploadAvatarUseCase
	getProfileUseCase    *usecase.GetProfileUseCase
}

// NewUserHandler Create a new UserHandler instance with dependencies
func NewUserHandler(
	updateProfileUseCase *usecase.UpdateProfileUseCase,
	uploadAvatarUseCase *usecase.UploadAvatarUseCase,
	getProfileUseCase *usecase.GetProfileUseCase,
) *UserHandler {
	return &UserHandler{
		updateProfileUseCase: updateProfileUseCase,
		uploadAvatarUseCase:  uploadAvatarUseCase,
		getProfileUseCase:    getProfileUseCase,
	}
}

// RegisterUserRoutes registers user-related API routes
func (h *UserHandler) RegisterUserRoutes(r *gin.RouterGroup) {
	r.GET("/:user_id", h.getProfile)
	r.PATCH("/profile", middleware.RequireAuthMiddleware(), h.updateProfile)
	r.POST("/avatar", middleware.RequireAuthMiddleware(), h.uploadAvatar)
}

// @Summary Get user profile
// @Description Get a user's public profile by ID
// @Tags User
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {object} GetUserProfileResponse
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 404 {object} response.ErrorResponse "Not Found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/user/{user_id} [get]
func (h *UserHandler) getProfile(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		HandleError(c, err)
		return
	}

	input := adapter.BuildInput(c, usecase.GetProfileInput{ID: id})
	out, err := h.getProfileUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, GetUserProfileResponse{
		ID:        out.ID,
		Name:      out.Name,
		Avatar:    out.Avatar,
		RoleCodes: out.RoleCodes,
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

	input := adapter.BuildInput(c, usecase.UpdateProfileInput{
		Name:      req.Name,
		RoleCodes: req.RoleCodes,
	})

	if _, err := h.updateProfileUseCase.Execute(c.Request.Context(), input); err != nil {
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

	// Limit the maximum size of the request body to 5 MB
	const maxSize = 5 << 20
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

	file, _, err := c.Request.FormFile("avatar")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			HandleError(c, user.ErrImageTooLarge)
			return
		}
		HandleError(c, err)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Log.Error("failed to close avatar file", zap.Error(err))
		}
	}()

	// Detect MIME type from the first 512 bytes, then seek back to start
	buf := make([]byte, 512)
	n, readErr := file.Read(buf)
	if readErr != nil && readErr != io.EOF {
		HandleError(c, user.ErrInvalidImageType)
		return
	}
	if !isAllowedImageType(http.DetectContentType(buf[:n])) {
		HandleError(c, user.ErrInvalidImageType)
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		HandleError(c, err)
		return
	}

	input := adapter.BuildInput(c, usecase.UploadAvatarInput{Image: file})
	output, err := h.uploadAvatarUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, UploadAvatarResponse{AvatarPath: output.AvatarPath})
}

func isAllowedImageType(mimeType string) bool {
	switch mimeType {
	case "image/jpeg", "image/png", "image/webp":
		return true
	}
	return false
}

package user

import (
	"errors"
	"net/http"

	"github.com/HiroLiang/goat-server/internal/domain/user"
	"github.com/HiroLiang/goat-server/internal/interface/http/response"
	"github.com/HiroLiang/goat-server/internal/logger"
	"github.com/gin-gonic/gin"
)

func HandleError(c *gin.Context, err error) {
	logger.Log.Error(err.Error())
	switch {
	case errors.Is(err, user.ErrUserNotFound):
		c.JSON(http.StatusNotFound, response.ErrNotFound("user"))
		return

	case errors.Is(err, user.ErrUserAlreadyExists):
		c.JSON(http.StatusConflict, response.ErrorResponse{
			Code:    "USER_EXIST",
			Message: "user already exists",
		})
		return

	case errors.Is(err, user.ErrInvalidUser):
		c.JSON(http.StatusForbidden, response.ErrInvalid("user"))
		return

	case errors.Is(err, user.ErrUserApplying):
		c.JSON(http.StatusForbidden, response.ErrorResponse{
			Code:    "USER_APPLYING",
			Message: "your registration is pending approval",
		})
		return

	case errors.Is(err, user.ErrUserBanned):
		c.JSON(http.StatusForbidden, response.ErrorResponse{
			Code:    "USER_BANNED",
			Message: "this account has been banned",
		})
		return

	case errors.Is(err, user.ErrInvalidPassword):
		c.JSON(http.StatusUnauthorized, response.ErrInvalid("password"))
		return

	case errors.Is(err, user.ErrInvalidEmail):
		c.JSON(http.StatusBadRequest, response.ErrInvalid("email"))
		return

	case errors.Is(err, user.ErrGenerateToken):
		c.JSON(http.StatusInternalServerError, response.ErrAuthFailed)
		return

	case errors.Is(err, user.ErrInvalidImageType):
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    "INVALID_IMAGE_TYPE",
			Message: "unsupported image type, allowed: jpeg, png, webp",
		})
		return

	case errors.Is(err, user.ErrImageTooLarge):
		c.JSON(http.StatusRequestEntityTooLarge, response.ErrorResponse{
			Code:    "IMAGE_TOO_LARGE",
			Message: "image exceeds maximum allowed size of 5 MB",
		})
		return

	default:
		_ = c.Error(err)
		return
	}
}

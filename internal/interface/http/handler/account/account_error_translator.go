package account

import (
	"errors"
	"net/http"

	"github.com/HiroLiang/goat-server/internal/application/auth/usecase"
	"github.com/HiroLiang/goat-server/internal/interface/http/response"
	"github.com/HiroLiang/goat-server/internal/logger"
	"github.com/gin-gonic/gin"
)

func HandleError(ctx *gin.Context, err error) {
	type errResponse struct {
		status  int
		code    string
		message string
	}

	errMap := []struct {
		target error
		res    errResponse
	}{
		{usecase.ErrRegisterFailed, errResponse{http.StatusInternalServerError, "REGISTER_FAILED", "register failed"}},
		{usecase.ErrLoginFailed, errResponse{http.StatusInternalServerError, "LOGIN_FAILED", "login failed"}},
		{usecase.ErrInvalidDeviceID, errResponse{http.StatusBadRequest, "INVALID_DEVICE_ID", "invalid device id"}},
		{usecase.ErrInvalidPassword, errResponse{http.StatusBadRequest, "INVALID_PASSWORD", "invalid password"}},
		{usecase.ErrInvalidEmail, errResponse{http.StatusBadRequest, "INVALID_EMAIL", "invalid email"}},
		{usecase.ErrEmailExist, errResponse{http.StatusConflict, "EMAIL_EXIST", "email already exists"}},
		{usecase.ErrAccountExist, errResponse{http.StatusConflict, "ACCOUNT_EXIST", "account already exists"}},
		{usecase.ErrAccountNotFound, errResponse{http.StatusNotFound, "ACCOUNT_NOT_FOUND", "account not found"}},
		{usecase.ErrAccountBanned, errResponse{http.StatusForbidden, "ACCOUNT_BANNED", "account has been banned"}},
		{usecase.ErrAccountApplying, errResponse{http.StatusForbidden, "ACCOUNT_APPLYING", "account is pending approval"}},
		{usecase.ErrAccountInactive, errResponse{http.StatusForbidden, "ACCOUNT_INACTIVE", "account is inactive"}},
		{usecase.ErrUserNotFound, errResponse{http.StatusNotFound, "USER_NOT_FOUND", "user not found"}},
		{usecase.ErrPasswordError, errResponse{http.StatusUnauthorized, "PASSWORD_ERROR", "incorrect password"}},
		{usecase.ErrTokenInvalid, errResponse{http.StatusBadRequest, "TOKEN_INVALID", "invalid or expired verification token"}},
	}

	for _, e := range errMap {
		if errors.Is(err, e.target) {
			ctx.JSON(e.res.status, response.ErrorResponse{
				Code:    e.res.code,
				Message: e.res.message,
			})
			return
		}
	}

	logger.Log.Error(err.Error())
	ctx.JSON(http.StatusInternalServerError, response.ErrorResponse{
		Code:    "INTERNAL_SERVER_ERROR",
		Message: "internal server error",
	})
}

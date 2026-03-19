package middleware

import (
	"errors"
	"net/http"

	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/response"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		// Catch first error
		err := c.Errors[0].Err

		// If is ErrorResponse
		var resp response.ErrorResponse
		if errors.As(err, &resp) {
			status := statusFromCode(resp.Code)
			c.JSON(status, gin.H{"error": resp})
			return
		}

		// Unknown error：500 Internal Server Error
		logger.Log.Error("Unknown error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": response.ErrorResponse{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
	}
}

func statusFromCode(code string) int {
	switch code {
	case "NOT_FOUND":
		return http.StatusNotFound
	case "INVALID":
		return http.StatusBadRequest
	case "AUTH_FAILED":
		return http.StatusUnauthorized
	case "RATE_LIMIT_EXCEEDED":
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

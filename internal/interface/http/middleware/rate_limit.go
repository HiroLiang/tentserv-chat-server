package middleware

import (
	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared/security"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/response"
	"github.com/gin-gonic/gin"
)

func GlobalRateLimitMiddleware(limiter security.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := limiter.CheckGlobal(c.Request.Context()); err != nil {
			_ = c.Error(response.ErrorResponse{
				Code:    "RATE_LIMIT_EXCEEDED",
				Message: "Server is busy, please try again later",
			})
			c.Abort()
			return
		}
		c.Next()

	}
}

func IPRateLimitMiddleware(limiter security.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if err := limiter.CheckIP(c.Request.Context(), ip); err != nil {
			_ = c.Error(response.ErrorResponse{
				Code:    "RATE_LIMIT_EXCEEDED",
				Message: "Too many requests from your IP, please try again later",
			})
			c.Abort()
			return
		}
		c.Next()

	}
}

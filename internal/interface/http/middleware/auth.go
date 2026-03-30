package middleware

import (
	"net/http"
	"strings"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/auth"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/response"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware try to validate auth token from the header
func AuthMiddleware(sessionManager port.SessionManager, userRepo user.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Get auth token from the header, falling back to query param for WebSocket
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			if t := c.Query("token"); t != "" {
				authHeader = "Bearer " + t
			}
		}
		DeviceID := c.GetHeader("X-Device-ID")

		// Validate token if exists
		var token auth.AccessToken = ""
		var validSession *auth.Session
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = auth.AccessToken(strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer ")))

			// Verify and get current session by token
			session, err := sessionManager.FindByToken(c.Request.Context(), token)
			if err == nil && session.DeviceID.Equal(DeviceID) {
				validSession = session

				// Find current user
				if userData, err := userRepo.FindByID(c.Request.Context(), session.UserID); err == nil {
					c.Set(AuthContextKey, &shared.AuthContext{
						AccountID:   session.AccountID,
						UserID:      session.UserID,
						Roles:       userData.RoleCodes,
						AccessToken: token,
					})
				}
			}
		}

		c.Next()

		// Set auth header to the response
		if validSession != nil {
			c.Header("Authorization", string(validSession.Token.AccessToken))
		}
	}
}

// RequireAuthMiddleware require auth context to be set or abort with 401
func RequireAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Check if auth context exists
		if _, exists := c.Get(AuthContextKey); !exists {
			logger.Log.Debug("authContext not exists")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": response.ErrAuthFailed,
			})
			return
		}
		c.Next()
	}
}

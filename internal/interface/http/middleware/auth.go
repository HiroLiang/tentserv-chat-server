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
	"go.uber.org/zap"
)

// [EN] AuthMiddleware: extracts Bearer token and X-Device-ID (falls back to query params for WebSocket),
//      validates the session, checks device ID matches, fetches user roles, and stores an AuthContext in Gin context.
//      Does NOT abort on failure — downstream RequireAuthMiddleware() handles that.
//      On success, echoes the refreshed token in the response Authorization header.
// [中] AuthMiddleware：提取 Bearer token 與 X-Device-ID（WebSocket 回退到 query param），
//      驗證 session、比對 device ID、取得使用者角色，並將 AuthContext 存入 Gin context。
//      驗證失敗不中止請求，由下游 RequireAuthMiddleware() 處理；成功時在回應 header 回傳更新後的 token。
// [日] AuthMiddleware：Bearer トークンと X-Device-ID を取得（WebSocket はクエリパラメータにフォールバック）、
//      セッションを検証し、デバイス ID を照合し、ユーザーロールを取得して AuthContext を Gin コンテキストに保存する。
//      失敗しても中断しない（下流の RequireAuthMiddleware() が担当）。成功時はレスポンスヘッダに更新トークンを返す。

// AuthMiddleware try to validate auth token from the header
func AuthMiddleware(sessionManager port.SessionManager, userRepo user.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		isWebSocketRequest := strings.HasPrefix(c.Request.URL.Path, "/ws")

		// Get auth token from the header, falling back to query param for WebSocket
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			if t := c.Query("token"); t != "" {
				authHeader = "Bearer " + t
			}
		}
		deviceID := c.GetHeader("X-Device-ID")
		if deviceID == "" {
			deviceID = c.Query("device_id") // WebSocket fallback (browser cannot set custom headers)
		}

		// Validate token if exists
		var token auth.AccessToken = ""
		var validSession *auth.Session
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = auth.AccessToken(strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer ")))

			// Verify and get current session by token
			session, err := sessionManager.FindByToken(c.Request.Context(), token)
			if err != nil {
				if isWebSocketRequest {
					logger.Log.Debug("websocket auth failed: session lookup failed",
						zap.String("path", c.Request.URL.Path),
						zap.String("device_id", deviceID),
						zap.Error(err),
					)
				}
			} else if !session.DeviceID.Equal(deviceID) {
				if isWebSocketRequest {
					logger.Log.Debug("websocket auth failed: device_id mismatch",
						zap.String("path", c.Request.URL.Path),
						zap.String("device_id", deviceID),
						zap.String("session_device_id", session.DeviceID.String()),
					)
				}
			} else {
				validSession = session

				// Find current user
				if userData, err := userRepo.FindByID(c.Request.Context(), session.UserID); err == nil {
					c.Set(AuthContextKey, &shared.AuthContext{
						AccountID:   session.AccountID,
						UserID:      session.UserID,
						Roles:       userData.RoleCodes,
						AccessToken: token,
					})
				} else if isWebSocketRequest {
					logger.Log.Debug("websocket auth failed: user lookup failed",
						zap.String("path", c.Request.URL.Path),
						zap.Int64("user_id", int64(session.UserID)),
						zap.Error(err),
					)
				}
			}
		} else if isWebSocketRequest {
			logger.Log.Debug("websocket auth failed: bearer token missing",
				zap.String("path", c.Request.URL.Path),
				zap.String("device_id", deviceID),
			)
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

package middleware

import (
	"net/http"

	appShared "github.com/HiroLiang/goat-server/internal/application/shared"
	"github.com/HiroLiang/goat-server/internal/domain/role"
	"github.com/HiroLiang/goat-server/internal/interface/http/response"
	"github.com/gin-gonic/gin"
)

// RequireRoleMiddleware aborts with 403 if the authenticated user does not hold any of the required roles.
func RequireRoleMiddleware(required ...role.Code) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, exists := c.Get(AuthContextKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrAuthFailed)
			return
		}

		authCtx, ok := v.(*appShared.AuthContext)
		if !ok || authCtx == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrAuthFailed)
			return
		}

		for _, userRole := range authCtx.Roles {
			for _, req := range required {
				if userRole == req {
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, response.ErrorResponse{
			Code:    "FORBIDDEN",
			Message: "insufficient permissions",
		})
	}
}

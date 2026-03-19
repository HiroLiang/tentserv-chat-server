package middleware

import (
	"net"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/gin-gonic/gin"
)

var AuthContextKey = "authContext"

// ContextMiddleware Build context data
func ContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Get auth context
		var authCtx *appShared.AuthContext
		if v, ok := c.Get(AuthContextKey); ok {
			authCtx = v.(*appShared.AuthContext)
		}

		did := c.GetHeader("X-Device-ID")
		deviceID, _ := shared.ParseDeviceID(did)

		// Set context
		c.Set("context", &appShared.BaseContext{
			Request: appShared.RequestContext{
				IP:       net.ParseIP(c.ClientIP()),
				TraceID:  c.GetHeader("traceparent"),
				DeviceID: deviceID,
			},
			Auth: authCtx,
		})

		c.Next()
	}
}

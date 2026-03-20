package middleware

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func AccessLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		ip := c.ClientIP()

		logger.Log.Info("→ request",
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", ip),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int64("content_length", c.Request.ContentLength),
		)

		c.Next()

		logger.Log.Info("← response",
			zap.String("method", method),
			zap.String("path", path),
			zap.String("ip", ip),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.Int("response_size", c.Writer.Size()),
		)
	}
}

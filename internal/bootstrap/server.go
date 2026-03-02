package bootstrap

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/HiroLiang/goat-server/internal/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewServer(addr string, useCases *UseCases, dependencies *Dependencies) *http.Server {
	r := gin.New()
	r.Use(
		gin.LoggerWithFormatter(func(params gin.LogFormatterParams) string {
			path := params.Path
			if strings.HasPrefix(path, "/ws") {
				path = "/ws?token=[REDACTED]"
			}

			statusColor := params.StatusCodeColor()
			methodColor := params.MethodColor()
			resetColor := params.ResetColor()

			return fmt.Sprintf("[GIN] %v |%s %3d %s| %13v | %15s |%s %-7s %s %s\n",
				params.TimeStamp.Format("2006/01/02 - 15:04:05"),
				statusColor, params.StatusCode, resetColor,
				params.Latency,
				params.ClientIP,
				methodColor, params.Method, resetColor,
				path,
			)
		}),
		gin.Recovery())

	// Init config
	initConfig(r)

	// Register Swagger
	if config.Env("APP_ENV", "dev") == "dev" {
		RegisterSwaggerRoutes(r.Group("/swagger"))
	}

	// Serve uploaded static files (avatars, etc.)
	r.Static("/static", config.App().Storage.LocalPath)

	// Register REST routes
	RegisterRestRoutes(r.Group("/api"), useCases, dependencies)

	// Register WebSocket routes
	hub, wsRouter := BuildWsComponents(dependencies)
	RegisterWsRoutes(r, hub, wsRouter, dependencies)

	// Setting server
	return &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   0,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}

func initConfig(r *gin.Engine) {

	// setting cors
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {

			if strings.HasSuffix(origin, "hiroliang.com") {
				return true
			}

			if strings.HasPrefix(origin, "http://localhost") {
				return true
			}

			if strings.HasPrefix(origin, "tauri://localhost") {
				return true
			}

			return false
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Accept", "Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Authorization", "X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
}

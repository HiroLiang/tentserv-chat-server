package bootstrap

import (
	"github.com/HiroLiang/goat-server/internal/config"
	"github.com/HiroLiang/goat-server/internal/interface/http/handler/agent"
	"github.com/HiroLiang/goat-server/internal/interface/http/handler/chat"
	"github.com/HiroLiang/goat-server/internal/interface/http/handler/device"
	"github.com/HiroLiang/goat-server/internal/interface/http/handler/health"
	"github.com/HiroLiang/goat-server/internal/interface/http/handler/test"
	"github.com/HiroLiang/goat-server/internal/interface/http/handler/user"
	"github.com/HiroLiang/goat-server/internal/interface/http/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRestRoutes(group *gin.RouterGroup, useCases *UseCases, dependencies *Dependencies) {

	// Global middleware
	group.Use(middleware.ErrorHandler())
	group.Use(middleware.GlobalRateLimitMiddleware(dependencies.RateLimiter))
	group.Use(middleware.IPRateLimitMiddleware(dependencies.RateLimiter))
	group.Use(middleware.AuthMiddleware(dependencies.TokenService))
	group.Use(middleware.ContextMiddleware())

	// Test Handler
	if config.Env("APP_ENV", "dev") == "dev" {
		var testHandler = test.NewTestHandler()
		testHandler.RegisterTestRoutes(group.Group("/test"))
	}

	// Health Check Handler
	var healthHandler = health.NewHealthHandler()
	healthHandler.RegisterHealthRoues(group.Group("/health"))

	// User Handler
	var userHandler = user.NewUserHandler(useCases.UserUseCase)
	userHandler.RegisterUserRoutes(group.Group("/user"))

	// Agent Handler
	var agentHandler = agent.NewAgentHandler(useCases.AgentUseCase)
	agentHandler.RegisterAgentRoutes(group.Group("/agent", middleware.RequireAuthMiddleware()))

	// Chat Handler
	var chatHandler = chat.NewChatHandler(useCases.ChatUseCase)
	chatHandler.RegisterChatRoutes(group.Group("/chat", middleware.RequireAuthMiddleware()))

	// Device Handler
	var deviceHandler = device.NewDeviceHandler(useCases.DeviceUseCase)
	deviceHandler.RegisterDeviceRoutes(group.Group("/device"))
}

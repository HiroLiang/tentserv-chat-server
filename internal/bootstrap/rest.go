package bootstrap

import (
	"github.com/HiroLiang/goat-server/internal/interface/http/handler/account"
	"github.com/HiroLiang/goat-server/internal/interface/http/handler/health"
	"github.com/HiroLiang/goat-server/internal/interface/http/handler/user"
	"github.com/HiroLiang/goat-server/internal/interface/http/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRestRoutes(group *gin.RouterGroup, useCases *UseCases, dependencies *Dependencies) {

	// Global middleware
	group.Use(middleware.ErrorHandler())
	group.Use(middleware.GlobalRateLimitMiddleware(dependencies.RateLimiter))
	group.Use(middleware.IPRateLimitMiddleware(dependencies.RateLimiter))
	group.Use(middleware.ContextMiddleware())

	// Health Check Handler
	var healthHandler = health.NewHealthHandler()
	healthHandler.RegisterHealthRoues(group.Group("/health"))

	// Auth Handlers
	var authHandler = account.NewAuthHandler(
		useCases.RegisterUseCase,
		useCases.LoginUseCase,
		useCases.LogoutUseCase,
		useCases.GetAccountProfileUseCase)
	authHandler.RegisterAuthRoutes(group.Group("/auth"))

	// User Handler
	var userHandler = user.NewUserHandler(
		useCases.UpdateUserProfileUseCase,
		useCases.UploadAvatarUseCase)
	userHandler.RegisterUserRoutes(group.Group("/user"))

	// Agent Handler
	//var agentHandler = agent.NewAgentHandler(useCases.AgentUseCase)
	//agentHandler.RegisterAgentRoutes(group.Group("/agent", middleware.RequireAuthMiddleware()))

	// Chat Handler
	//var chatHandler = chat.NewChatHandler(useCases.ChatUseCase)
	//chatHandler.RegisterChatRoutes(group.Group("/chat", middleware.RequireAuthMiddleware()))

	// Device Handler
	//var deviceHandler = device.NewDeviceHandler(useCases.DeviceUseCase)
	//deviceHandler.RegisterDeviceRoutes(group.Group("/device"))

	// Participant Handler
	//var participantHandler = participant.NewParticipantHandler(useCases.ParticipantUseCase)
	//participantHandler.RegisterParticipantRoutes(group.Group("/participant", middleware.RequireAuthMiddleware()))
}

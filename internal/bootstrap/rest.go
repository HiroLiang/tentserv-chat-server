package bootstrap

import (
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/chat"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/e2ee"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/health"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRestRoutes(group *gin.RouterGroup, useCases *UseCases, dependencies *Dependencies) {

	// Global middleware
	group.Use(middleware.ErrorHandler())
	group.Use(middleware.GlobalRateLimitMiddleware(dependencies.RateLimiter))
	group.Use(middleware.IPRateLimitMiddleware(dependencies.RateLimiter))
	group.Use(middleware.AccessLogMiddleware())
	group.Use(middleware.AuthMiddleware(dependencies.SessionManager, dependencies.UserRepo))
	group.Use(middleware.ContextMiddleware())

	// Health Check Handler
	var healthHandler = health.NewHealthHandler()
	healthHandler.RegisterHealthRoues(group.Group("/health"))

	// Auth Handlers
	var authHandler = account.NewAuthHandler(
		useCases.RegisterUseCase,
		useCases.LoginUseCase,
		useCases.LogoutUseCase,
		useCases.GetAccountProfileUseCase,
		useCases.VerifyEmailUseCase)
	authHandler.RegisterAuthRoutes(group.Group("/auth"))

	// User Handler
	var userHandler = user.NewUserHandler(
		useCases.UpdateUserProfileUseCase,
		useCases.UploadAvatarUseCase,
		useCases.GetUserProfileUseCase)
	userHandler.RegisterUserRoutes(group.Group("/user"))

	var friendshipHandler = user.NewFriendshipHandler(useCases.GetFriendsUseCase)
	friendshipHandler.RegisterFriendshipRoutes(group.Group("/user", middleware.RequireAuthMiddleware()))

	var deviceHandler = device.NewDeviceHandler(
		useCases.RegisterDeviceUseCase,
		useCases.GetDeviceProfileUseCase,
		useCases.UpdateDeviceUseCase)
	deviceHandler.RegisterDeviceRoutes(group.Group("/device"))

	// Agent Handler
	//var agentHandler = agent.NewAgentHandler(useCases.AgentUseCase)
	//agentHandler.RegisterAgentRoutes(group.Group("/agent", middleware.RequireAuthMiddleware()))

	// Chat Handler
	chatGroup := group.Group("/chat", middleware.RequireAuthMiddleware())
	var chatRoomHandler = chat.NewChatRoomHandler(
		useCases.CreateChatRoomUseCase,
		useCases.JoinChatRoomUseCase,
		useCases.ApproveJoinRequestUseCase,
		useCases.GetUserChatRoomsUseCase,
		useCases.GetChatRoomDetailUseCase,
		useCases.GetChatRoomMessagesUseCase,
		useCases.UpdateMemberStatusUseCase,
		useCases.SendMessageUseCase,
		useCases.UploadRoomMediaUseCase,
		dependencies.LocalFileStorage)
	chatRoomHandler.RegisterChatRoomRoutes(chatGroup)

	// Participant Handler
	participantGroup := group.Group("/participant", middleware.RequireAuthMiddleware())
	var participantHandler = participant.NewParticipantHandler(
		useCases.CreateUserParticipantUseCase,
		useCases.GetUserParticipantUseCase)
	participantHandler.RegisterParticipantRoutes(participantGroup)

	// E2EE Handler
	e2eeGroup := group.Group("/e2ee", middleware.RequireAuthMiddleware())
	e2ee.NewE2EEHandler(
		useCases.UploadIdentityKeyUseCase,
		useCases.UploadSignedPreKeyUseCase,
		useCases.UploadOTPPreKeysUseCase,
		useCases.CountOTPPreKeysUseCase,
		useCases.GetKeyBundleUseCase,
		useCases.UploadSenderKeyUseCase,
		useCases.GetSenderKeysUseCase,
	).RegisterE2EERoutes(e2eeGroup)

	// Future: admin-only participant routes
	// adminGroup := group.Group("/admin/participant", middleware.RequireAuthMiddleware(), middleware.RequireRoleMiddleware(role.Admin))
	// participantHandler.RegisterAdminParticipantRoutes(adminGroup)
}

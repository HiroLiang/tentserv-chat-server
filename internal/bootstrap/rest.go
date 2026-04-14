package bootstrap

import (
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/chat"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/e2ee"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/health"
	ollamaHandler "github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/ollama"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/middleware"
	"github.com/gin-gonic/gin"
)

// [EN] RegisterRestRoutes wires all REST routes onto the Gin router group.
//
//	Middleware stack (applied to all routes): ErrorHandler → GlobalRateLimit → IPRateLimit →
//	AccessLog → AuthMiddleware (sets authContext if token valid) → ContextMiddleware.
//	Protected route groups additionally use RequireAuthMiddleware() which aborts with 401 if no authContext.
//
// [中] RegisterRestRoutes 將所有 REST 路由掛載到 Gin RouterGroup。
//
//	中間件堆疊（全域）：ErrorHandler → GlobalRateLimit → IPRateLimit →
//	AccessLog → AuthMiddleware（Token 有效時設定 authContext）→ ContextMiddleware。
//	受保護路由群組額外使用 RequireAuthMiddleware()，無 authContext 時返回 401。
//
// [日] RegisterRestRoutes はすべての REST ルートを Gin RouterGroup に登録する。
//
//	ミドルウェアスタック（全ルート共通）：ErrorHandler → GlobalRateLimit → IPRateLimit →
//	AccessLog → AuthMiddleware（トークン有効時に authContext を設定）→ ContextMiddleware。
//	保護ルートグループはさらに RequireAuthMiddleware() を適用し、authContext なしの場合は 401 を返す。
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
		useCases.VerifyEmailUseCase,
		useCases.ResendVerifyEmailUseCase,
		useCases.VerifyLoginDeviceUseCase,
		useCases.ResendLoginDeviceVerificationUseCase,
		dependencies.RegisterRateLimiter)
	authHandler.RegisterAuthRoutes(group.Group("/auth"))

	// User Handler
	var userHandler = user.NewUserHandler(
		useCases.UpdateUserProfileUseCase,
		useCases.UploadAvatarUseCase,
		useCases.GetUserProfileUseCase,
		useCases.SearchUsersUseCase,
		useCases.SwitchUserUseCase)
	userHandler.RegisterUserRoutes(group.Group("/user"))

	var friendshipHandler = friendship.NewFriendshipHandler(
		useCases.GetFriendsUseCase,
		useCases.GetBlockedUsersUseCase,
		useCases.ApplyFriendshipUseCase,
		useCases.AcceptFriendshipUseCase,
		useCases.GetFriendRequestsUseCase,
		useCases.RemoveFriendshipUseCase,
		useCases.GetSentRequestsUseCase,
		useCases.CancelSentRequestUseCase,
		useCases.BlockUserUseCase,
		useCases.UnblockUserUseCase,
	)
	friendshipHandler.RegisterFriendshipRoutes(group.Group("/user", middleware.RequireAuthMiddleware()))

	var deviceHandler = device.NewDeviceHandler(
		useCases.RegisterDeviceUseCase,
		useCases.GetDeviceProfileUseCase,
		useCases.UpdateDeviceUseCase,
		useCases.ListDevicesUseCase,
		useCases.BindAccountUseCase,
		useCases.DeleteDeviceUseCase)
	deviceHandler.RegisterPublicDeviceRoutes(group.Group("/device"))
	deviceHandler.RegisterProtectedDeviceRoutes(group.Group("/device", middleware.RequireAuthMiddleware()))

	// Agent Handler
	//var agentHandler = agent.NewAgentHandler(useCases.AgentUseCase)
	//agentHandler.RegisterAgentRoutes(group.Group("/agent", middleware.RequireAuthMiddleware()))

	// Chat Handler
	chatGroup := group.Group("/chat", middleware.RequireAuthMiddleware())
	var chatRoomHandler = chat.NewChatRoomHandler(
		useCases.CreateChatRoomUseCase,
		useCases.JoinChatRoomUseCase,
		useCases.ApproveJoinRequestUseCase,
		useCases.GetMyRoomInvitationUseCase,
		useCases.RespondToInvitationUseCase,
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
		useCases.CheckKeyStatusUseCase,
		useCases.GetKeyPolicyUseCase,
		useCases.UploadSenderKeyUseCase,
		useCases.GetSenderKeysUseCase,
		useCases.GetSenderKeyDistributionStatusUseCase,
		useCases.GetPendingSenderKeyDistributionsUseCase,
		useCases.ConsumeSenderKeyDistributionUseCase,
		useCases.CreateSenderKeyRequestUseCase,
		useCases.UploadSelfSenderKeySyncDistributionsUseCase,
		useCases.GetPendingSelfSenderKeySyncDistributionsUseCase,
		useCases.ConsumeSelfSenderKeySyncDistributionUseCase,
		useCases.GetSelfSenderKeySyncUseCase,
		useCases.SelfSenderKeySyncMutationUseCase,
	).RegisterE2EERoutes(e2eeGroup)

	// Future: admin-only participant routes
	// adminGroup := group.Group("/admin/participant", middleware.RequireAuthMiddleware(), middleware.RequireRoleMiddleware(role.Admin))
	// participantHandler.RegisterAdminParticipantRoutes(adminGroup)

	// Ollama streaming proxy — protected by GlobalRateLimit + IPRateLimit from group middleware
	// Authentication uses X-Chat-Api-Key header (not JWT), so RequireAuthMiddleware is not applied
	var ollamaStreamHandler = ollamaHandler.NewOllamaChatHandler(useCases.StreamChatUseCase)
	group.GET("/ollama/stream", ollamaStreamHandler.Stream)
}

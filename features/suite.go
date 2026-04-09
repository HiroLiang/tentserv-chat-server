package features

import (
	"net/http/httptest"

	accountfeatures "github.com/HiroLiang/tentserv-chat-server/features/account"
	devicefeatures "github.com/HiroLiang/tentserv-chat-server/features/device"
	e2eefeatures "github.com/HiroLiang/tentserv-chat-server/features/e2ee"
	friendshipfeatures "github.com/HiroLiang/tentserv-chat-server/features/friendship"
	bddsupport "github.com/HiroLiang/tentserv-chat-server/features/support"
	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/shared/security"
	accountHandler "github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/account"
	deviceHandler "github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/device"
	e2eeHandler "github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/e2ee"
	friendshipHandler "github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/friendship"
	userHandler "github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/middleware"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/cucumber/godog"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	testServer    *httptest.Server
	baseURL       string
	accountBDD    *accountfeatures.Deps
	deviceBDD     *devicefeatures.Deps
	e2eeBDD       *e2eefeatures.Deps
	friendshipBDD *friendshipfeatures.Deps
)

func InitializeSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		gin.SetMode(gin.TestMode)
		if err := config.LoadConfig("../dev-doc/config"); err != nil {
			panic(err)
		}
		logger.Log = zap.NewNop()

		accountBDD = accountfeatures.NewDeps()
		deviceBDD = devicefeatures.NewDeps()
		e2eeBDD = e2eefeatures.NewDeps()
		friendshipBDD = friendshipfeatures.NewDeps()

		hasher := security.NewArgon2Hasher()
		accountRegisterUseCase, verifyEmailUseCase, resendVerifyEmailUseCase := accountBDD.RegisterUseCases(
			bddsupport.UOW{},
			hasher,
		)
		loginUseCase, logoutUseCase, getProfileUseCase := accountBDD.LoginUseCases(bddsupport.UOW{}, hasher)
		deviceRegisterUseCase, deviceUpdateUseCase := deviceBDD.RegisterUseCases(bddsupport.UOW{})
		e2eeUseCases := e2eeBDD.RegisterUseCases()
		createSKRUseCase := e2eeBDD.SKR.RegisterCreateSenderKeyRequestUseCase()
		friendshipUseCases := friendshipBDD.RegisterUseCases(bddsupport.UOW{})

		router := gin.New()
		router.Use(middleware.AuthMiddleware(accountBDD.SessionManager(), accountBDD.UserRepo()))
		router.Use(middleware.ContextMiddleware())
		authHandler := accountHandler.NewAuthHandler(accountRegisterUseCase, loginUseCase, logoutUseCase, getProfileUseCase, verifyEmailUseCase, resendVerifyEmailUseCase, accountBDD.RegisterLimiter())
		authHandler.RegisterAuthRoutes(router.Group("/api/auth"))
		deviceRoutes := deviceHandler.NewDeviceHandler(deviceRegisterUseCase, nil, deviceUpdateUseCase, nil, nil, nil)
		deviceRoutes.RegisterPublicDeviceRoutes(router.Group("/api/device"))
		userHandler.NewUserHandler(
			nil,
			nil,
			nil,
			friendshipUseCases.SearchUsers,
			nil,
		).RegisterUserRoutes(router.Group("/api/user"))
		friendshipHandler.NewFriendshipHandler(
			friendshipUseCases.GetFriends,
			friendshipUseCases.GetBlockedUsers,
			friendshipUseCases.ApplyFriendship,
			friendshipUseCases.AcceptFriendship,
			friendshipUseCases.GetFriendRequests,
			friendshipUseCases.RemoveFriendship,
			friendshipUseCases.GetSentRequests,
			friendshipUseCases.CancelSentRequest,
			friendshipUseCases.BlockUser,
			friendshipUseCases.UnblockUser,
		).RegisterFriendshipRoutes(router.Group("/api/user", middleware.RequireAuthMiddleware()))
		e2eeHandler.NewE2EEHandler(
			e2eeUseCases.UploadIdentityKey,
			e2eeUseCases.UploadSignedPreKey,
			e2eeUseCases.UploadOTPPreKeys,
			e2eeUseCases.CountOTPPreKeys,
			e2eeUseCases.GetKeyBundle,
			e2eeUseCases.CheckKeyStatus,
			e2eeUseCases.GetKeyPolicy,
			nil,
			nil,
			nil,
			createSKRUseCase,
		).RegisterE2EERoutes(router.Group("/api/e2ee", middleware.RequireAuthMiddleware()))

		testServer = httptest.NewServer(router)
		baseURL = testServer.URL
	})

	ctx.AfterSuite(func() {
		if testServer != nil {
			testServer.Close()
		}
	})
}

package features

import (
	"net/http/httptest"

	accountfeatures "github.com/HiroLiang/tentserv-chat-server/features/account"
	devicefeatures "github.com/HiroLiang/tentserv-chat-server/features/device"
	bddsupport "github.com/HiroLiang/tentserv-chat-server/features/support"
	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/shared/security"
	accountHandler "github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/account"
	deviceHandler "github.com/HiroLiang/tentserv-chat-server/internal/interface/http/handler/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/middleware"
	"github.com/cucumber/godog"
	"github.com/gin-gonic/gin"
)

var (
	testServer *httptest.Server
	baseURL    string
	accountBDD *accountfeatures.Deps
	deviceBDD  *devicefeatures.Deps
)

func InitializeSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		gin.SetMode(gin.TestMode)
		if err := config.LoadConfig("../dev-doc/config"); err != nil {
			panic(err)
		}

		accountBDD = accountfeatures.NewDeps()
		deviceBDD = devicefeatures.NewDeps()

		accountRegisterUseCase, verifyEmailUseCase := accountBDD.RegisterUseCases(
			bddsupport.UOW{},
			security.NewArgon2Hasher(),
		)
		deviceRegisterUseCase, deviceUpdateUseCase := deviceBDD.RegisterUseCases(bddsupport.UOW{})

		router := gin.New()
		router.Use(middleware.ContextMiddleware())
		authHandler := accountHandler.NewAuthHandler(accountRegisterUseCase, nil, nil, nil, verifyEmailUseCase)
		authHandler.RegisterAuthRoutes(router.Group("/api/auth"))
		deviceRoutes := deviceHandler.NewDeviceHandler(deviceRegisterUseCase, nil, deviceUpdateUseCase, nil, nil, nil)
		deviceRoutes.RegisterPublicDeviceRoutes(router.Group("/api/device"))

		testServer = httptest.NewServer(router)
		baseURL = testServer.URL
	})

	ctx.AfterSuite(func() {
		if testServer != nil {
			testServer.Close()
		}
	})
}

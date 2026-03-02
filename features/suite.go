package features

import (
	"fmt"
	"net/http/httptest"

	"github.com/HiroLiang/goat-server/internal/bootstrap"
	"github.com/HiroLiang/goat-server/internal/config"
	mockAuth "github.com/HiroLiang/goat-server/internal/infrastructure/auth/mock"
	"github.com/HiroLiang/goat-server/internal/infrastructure/auth/token"
	mockRepo "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/mock"
	mockShared "github.com/HiroLiang/goat-server/internal/infrastructure/shared/mock"
	"github.com/HiroLiang/goat-server/internal/logger"
	"github.com/cucumber/godog"
	"go.uber.org/zap"
)

var (
	testServer *httptest.Server
	baseURL    string
)

// InitializeSuite runs once before all features
func InitializeSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		logger.InitTestEnv()

		if err := config.LoadConfig("../dev-doc/config"); err != nil {
			logger.Log.Fatal("Error loading config", zap.Error(err))
		}

		conf := config.App()

		// Get mock dependencies
		sessionStore := mockAuth.MockSessionStore()
		dependencies := bootstrap.MockDeps(
			func(deps *bootstrap.Dependencies) {
				deps.AgentRepo = mockRepo.MockAgentRepository() // TODO not implemented
				deps.FileStorage = mockShared.MockFileStorage()
				deps.TokenService = token.NewAuthTokenService(sessionStore, conf.AuthToken.Expiration)
				deps.RateLimiter = mockShared.MockRateLimiter()
				deps.UserRepo = mockRepo.MockUserRepo()         // TODO not implemented
				deps.UserRoleRepo = mockAuth.MockUserRoleRepo() // TODO not implemented
			},
		)

		// Build use cases
		useCases := bootstrap.BuildUseCases(dependencies)

		// For feature tests we only hit /api/test, so pass empty services
		app := bootstrap.NewServer(":8080", useCases, dependencies)
		testServer = httptest.NewServer(app.Handler)
		baseURL = testServer.URL
		fmt.Println(testServer.URL)
	})
	ctx.AfterSuite(func() {
		testServer.Close()
		logger.Stop()
	})
}

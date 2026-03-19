package features

import (
	"fmt"
	"net/http/httptest"

	"github.com/HiroLiang/tentserv-chat-server/internal/bootstrap"
	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	mockAuth "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/auth/mock"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
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

		// Get mock dependencies
		dependencies := bootstrap.MockDeps(
			func(deps *bootstrap.Dependencies) {
				deps.UserRoleRepo = mockAuth.MockUserRoleRepo()
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

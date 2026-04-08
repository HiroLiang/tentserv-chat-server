package features

import (
	"context"

	accountfeatures "github.com/HiroLiang/tentserv-chat-server/features/account"
	devicefeatures "github.com/HiroLiang/tentserv-chat-server/features/device"
	bddsupport "github.com/HiroLiang/tentserv-chat-server/features/support"
	"github.com/cucumber/godog"
)

func InitializeScenario(ctx *godog.ScenarioContext) {
	apiCtx := bddsupport.NewAPITestContext(baseURL)

	ctx.Before(func(scenarioCtx context.Context, _ *godog.Scenario) (context.Context, error) {
		accountBDD.Reset()
		deviceBDD.Reset()
		apiCtx.Reset()
		return scenarioCtx, nil
	})

	bddsupport.RegisterCommonSteps(ctx, apiCtx)
	accountfeatures.RegisterSteps(ctx, apiCtx, accountBDD)
	devicefeatures.RegisterSteps(ctx, apiCtx, deviceBDD)
}

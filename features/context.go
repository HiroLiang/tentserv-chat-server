package features

import (
	"context"

	accountfeatures "github.com/HiroLiang/tentserv-chat-server/features/account"
	devicefeatures "github.com/HiroLiang/tentserv-chat-server/features/device"
	e2eefeatures "github.com/HiroLiang/tentserv-chat-server/features/e2ee"
	friendshipfeatures "github.com/HiroLiang/tentserv-chat-server/features/friendship"
	integrationfeatures "github.com/HiroLiang/tentserv-chat-server/features/integration"
	bddsupport "github.com/HiroLiang/tentserv-chat-server/features/support"
	"github.com/cucumber/godog"
)

func InitializeScenario(ctx *godog.ScenarioContext) {
	apiCtx := bddsupport.NewAPITestContext(baseURL)

	ctx.Before(func(scenarioCtx context.Context, _ *godog.Scenario) (context.Context, error) {
		accountBDD.Reset()
		deviceBDD.Reset()
		e2eeBDD.Reset()
		friendshipBDD.Reset()
		apiCtx.Reset()
		return scenarioCtx, nil
	})

	bddsupport.RegisterCommonSteps(ctx, apiCtx)
	accountfeatures.RegisterSteps(ctx, apiCtx, accountBDD)
	devicefeatures.RegisterSteps(ctx, apiCtx, deviceBDD)
	e2eefeatures.RegisterSteps(ctx, apiCtx, e2eeBDD, accountBDD)
	friendshipfeatures.RegisterSteps(ctx, apiCtx, friendshipBDD, accountBDD)
	integrationfeatures.RegisterSteps(ctx, apiCtx, accountBDD, friendshipBDD)
}

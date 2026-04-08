package features

import (
	"testing"

	"github.com/cucumber/godog"
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		TestSuiteInitializer: InitializeSuite,
		ScenarioInitializer:  InitializeScenario,
		Options: &godog.Options{
			Paths:  []string{"account", "device", "e2ee", "friendship"},
			Format: "pretty",
		},
	}
	if suite.Run() != 0 {
		t.Fail()
	}
}

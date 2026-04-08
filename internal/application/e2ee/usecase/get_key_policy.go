package usecase

import (
	"context"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/config"
)

type GetKeyPolicyInput struct{}

type GetKeyPolicyOutput struct {
	OTPPreKeyTargetCount        int
	OTPPreKeyReplenishThreshold int
}

type keyPolicyConfig struct {
	OTPPreKeyTargetCount        int
	OTPPreKeyReplenishThreshold int
}

type GetKeyPolicyUseCase struct {
	loadConfig func() keyPolicyConfig
}

func NewGetKeyPolicyUseCase() *GetKeyPolicyUseCase {
	return newGetKeyPolicyUseCase(loadAppKeyPolicyConfig)
}

func newGetKeyPolicyUseCase(loadConfig func() keyPolicyConfig) *GetKeyPolicyUseCase {
	return &GetKeyPolicyUseCase{loadConfig: loadConfig}
}

func (u *GetKeyPolicyUseCase) Execute(
	context.Context,
	appShared.UseCaseInput[GetKeyPolicyInput],
) (*GetKeyPolicyOutput, error) {
	target, threshold := normalizeKeyPolicyConfig(u.loadConfig())
	return &GetKeyPolicyOutput{
		OTPPreKeyTargetCount:        target,
		OTPPreKeyReplenishThreshold: threshold,
	}, nil
}

func loadAppKeyPolicyConfig() keyPolicyConfig {
	conf := config.App()
	return keyPolicyConfig{
		OTPPreKeyTargetCount:        conf.E2EE.OTPPreKeyTargetCount,
		OTPPreKeyReplenishThreshold: conf.E2EE.OTPPreKeyReplenishThreshold,
	}
}

func normalizeKeyPolicyConfig(policy keyPolicyConfig) (target int, threshold int) {
	target = policy.OTPPreKeyTargetCount
	if target <= 0 {
		target = DefaultOTPPreKeyTargetCount
	}
	threshold = policy.OTPPreKeyReplenishThreshold
	if threshold <= 0 {
		threshold = DefaultOTPPreKeyReplenishThreshold
	}
	return target, threshold
}

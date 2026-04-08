package usecase

import (
	"context"
	"testing"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetKeyPolicyUseCase_ReturnsConfiguredPolicy(t *testing.T) {
	uc := newGetKeyPolicyUseCase(func() keyPolicyConfig {
		return keyPolicyConfig{
			OTPPreKeyTargetCount:        30,
			OTPPreKeyReplenishThreshold: 8,
		}
	})

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[GetKeyPolicyInput]{})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, 30, out.OTPPreKeyTargetCount)
	assert.Equal(t, 8, out.OTPPreKeyReplenishThreshold)
}

func TestGetKeyPolicyUseCase_UsesDefaultsWhenConfigIsUnset(t *testing.T) {
	uc := newGetKeyPolicyUseCase(func() keyPolicyConfig {
		return keyPolicyConfig{}
	})

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[GetKeyPolicyInput]{})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, DefaultOTPPreKeyTargetCount, out.OTPPreKeyTargetCount)
	assert.Equal(t, DefaultOTPPreKeyReplenishThreshold, out.OTPPreKeyReplenishThreshold)
}

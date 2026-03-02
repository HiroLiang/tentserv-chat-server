package bootstrap

import (
	"github.com/HiroLiang/goat-server/internal/application/agent"
	"github.com/HiroLiang/goat-server/internal/application/chat"
	appdevice "github.com/HiroLiang/goat-server/internal/application/device"
	"github.com/HiroLiang/goat-server/internal/application/user"
	"github.com/HiroLiang/goat-server/internal/config"
)

type UseCases struct {
	UserUseCase   *user.UseCase
	AgentUseCase  *agent.UseCase
	ChatUseCase   *chat.UseCase
	DeviceUseCase *appdevice.UseCase
}

func BuildUseCases(deps *Dependencies) *UseCases {

	// get config
	conf := config.App()

	return &UseCases{
		UserUseCase: user.NewUseCase(
			conf.Storage.BaseURL,
			deps.UserRepo,
			deps.UserRoleRepo,
			deps.Argon2Hasher,
			deps.TokenService,
			deps.FileStorage,
			deps.DeviceRepo,
		),
		AgentUseCase: agent.NewUseCase(deps.AgentRepo),
		ChatUseCase: chat.NewUseCase(
			deps.ParticipantRepo,
			deps.ChatGroupRepo,
			deps.ChatMemberRepo,
			deps.ChatMessageRepo,
		),
		DeviceUseCase: appdevice.NewUseCase(deps.DeviceRepo),
	}
}

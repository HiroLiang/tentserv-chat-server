package bootstrap

import (
	authUseCase "github.com/HiroLiang/goat-server/internal/application/auth/usecase"
	deviceUseCase "github.com/HiroLiang/goat-server/internal/application/device/usecase"
	userUseCase "github.com/HiroLiang/goat-server/internal/application/user/usecase"
)

type UseCases struct {
	RegisterUseCase          *authUseCase.RegisterUseCase
	LoginUseCase             *authUseCase.LoginUseCase
	LogoutUseCase            *authUseCase.LogoutUseCase
	GetAccountProfileUseCase *authUseCase.GetProfileUseCase

	UpdateUserProfileUseCase *userUseCase.UpdateProfileUseCase
	UploadAvatarUseCase      *userUseCase.UploadAvatarUseCase

	RegisterDeviceUseCase   *deviceUseCase.RegisterUseCase
	GetDeviceProfileUseCase *deviceUseCase.GetProfileUseCase
	UpdateDeviceUseCase     *deviceUseCase.UpdateDeviceUseCase
}

func BuildUseCases(deps *Dependencies) *UseCases {
	return &UseCases{
		RegisterUseCase:          authUseCase.NewRegisterUseCase(deps.Uow, deps.PwdHasher, deps.AccountRepo, deps.UserRepo),
		LoginUseCase:             authUseCase.NewLoginUseCase(deps.Uow, deps.PwdHasher, deps.SessionManager, deps.AccountRepo, deps.UserRepo),
		LogoutUseCase:            authUseCase.NewLogoutUseCase(deps.SessionManager),
		GetAccountProfileUseCase: authUseCase.NewGetProfileUseCase(deps.AccountRepo, deps.UserRepo),

		UpdateUserProfileUseCase: userUseCase.NewUpdateProfileUseCase(deps.UserRepo),
		UploadAvatarUseCase:      userUseCase.NewUploadAvatarUseCase(deps.ContextHasher, deps.LocalFileStorage, deps.UserRepo),

		RegisterDeviceUseCase:   deviceUseCase.NewRegisterUseCase(deps.Uow, deps.DeviceRepo),
		GetDeviceProfileUseCase: deviceUseCase.NewGetProfileUseCase(deps.Uow, deps.DeviceRepo),
		UpdateDeviceUseCase:     deviceUseCase.NewUpdateDeviceUseCase(deps.DeviceRepo),
	}
}

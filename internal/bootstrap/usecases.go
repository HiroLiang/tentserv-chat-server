package bootstrap

import (
	"time"

	authUseCase "github.com/HiroLiang/goat-server/internal/application/auth/usecase"
	deviceUseCase "github.com/HiroLiang/goat-server/internal/application/device/usecase"
	appEmail "github.com/HiroLiang/goat-server/internal/application/shared/email"
	userUseCase "github.com/HiroLiang/goat-server/internal/application/user/usecase"
	"github.com/HiroLiang/goat-server/internal/config"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
	infraBuilder "github.com/HiroLiang/goat-server/internal/infrastructure/email/builder"
)

type UseCases struct {
	RegisterUseCase          *authUseCase.RegisterUseCase
	LoginUseCase             *authUseCase.LoginUseCase
	LogoutUseCase            *authUseCase.LogoutUseCase
	GetAccountProfileUseCase *authUseCase.GetProfileUseCase
	VerifyEmailUseCase       *authUseCase.VerifyEmailUseCase

	UpdateUserProfileUseCase *userUseCase.UpdateProfileUseCase
	UploadAvatarUseCase      *userUseCase.UploadAvatarUseCase
	GetUserProfileUseCase    *userUseCase.GetProfileUseCase

	RegisterDeviceUseCase   *deviceUseCase.RegisterUseCase
	GetDeviceProfileUseCase *deviceUseCase.GetProfileUseCase
	UpdateDeviceUseCase     *deviceUseCase.UpdateDeviceUseCase
}

func BuildUseCases(deps *Dependencies) *UseCases {
	conf := config.App()
	sender := shared.EmailSender{
		Address: shared.EmailAddress(conf.Email.SenderAddress),
		Name:    conf.Email.SenderName,
	}

	return &UseCases{
		RegisterUseCase: authUseCase.NewRegisterUseCase(
			deps.Uow,
			deps.PwdHasher,
			deps.AccountRepo,
			deps.UserRepo,
			deps.VerificationStore,
			deps.EmailService,
			func(recipientEmail, recipientName, verifyURL string) appEmail.EmailBuilder {
				return infraBuilder.NewRegisterMailBuilder(sender, recipientEmail, recipientName, verifyURL)
			},
		),
		LoginUseCase: authUseCase.NewLoginUseCase(
			deps.Uow, deps.PwdHasher, deps.SessionManager,
			deps.AccountRepo, deps.UserRepo,
			deps.EmailService,
			func(recipientEmail, recipientName, deviceID, ip string, loginTime time.Time) appEmail.EmailBuilder {
				return infraBuilder.NewLoginMailBuilder(sender, recipientEmail, recipientName, deviceID, ip, loginTime)
			},
		),
		LogoutUseCase:            authUseCase.NewLogoutUseCase(deps.SessionManager),
		GetAccountProfileUseCase: authUseCase.NewGetProfileUseCase(deps.AccountRepo, deps.UserRepo),
		VerifyEmailUseCase:       authUseCase.NewVerifyEmailUseCase(deps.VerificationStore, deps.AccountRepo),

		UpdateUserProfileUseCase: userUseCase.NewUpdateProfileUseCase(deps.UserRepo),
		UploadAvatarUseCase:      userUseCase.NewUploadAvatarUseCase(deps.ContextHasher, deps.LocalFileStorage, deps.UserRepo),
		GetUserProfileUseCase:    userUseCase.NewGetProfileUseCase(deps.UserRepo),

		RegisterDeviceUseCase:   deviceUseCase.NewRegisterUseCase(deps.Uow, deps.DeviceRepo),
		GetDeviceProfileUseCase: deviceUseCase.NewGetProfileUseCase(deps.Uow, deps.DeviceRepo),
		UpdateDeviceUseCase:     deviceUseCase.NewUpdateDeviceUseCase(deps.DeviceRepo),
	}
}

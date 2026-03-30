package usecase

import (
	"context"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/shared/timeutil"
)

type ListDevicesOutput struct {
	DeviceID  string
	Name      string
	Platform  string
	CreatedAt string
}

type ListDevicesUseCase struct {
	deviceRepo device.Repository
}

func NewListDevicesUseCase(deviceRepo device.Repository) *ListDevicesUseCase {
	return &ListDevicesUseCase{deviceRepo: deviceRepo}
}

func (uc *ListDevicesUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[struct{}],
) ([]ListDevicesOutput, error) {
	if input.Base.Auth == nil {
		return nil, ErrUnauthorized
	}

	devices, err := uc.deviceRepo.FindAllByAccountID(ctx, input.Base.Auth.AccountID)
	if err != nil {
		return nil, err
	}

	out := make([]ListDevicesOutput, 0, len(devices))
	for _, d := range devices {
		out = append(out, ListDevicesOutput{
			DeviceID:  d.ID.String(),
			Name:      d.Name,
			Platform:  string(d.Platform),
			CreatedAt: timeutil.Format(d.CreatedAt, timeutil.FormatISO),
		})
	}
	return out, nil
}

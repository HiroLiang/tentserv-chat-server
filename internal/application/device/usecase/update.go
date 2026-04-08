package usecase

import (
	"context"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/shared/timeutil"
)

type UpdateDeviceInput struct {
	DeviceID string
	Name     string
	Platform string
}

type UpdateDeviceOutput struct {
	DeviceID  string
	Name      string
	Platform  string
	CreatedAt string
	UpdatedAt string
}

type UpdateDeviceUseCase struct {
	deviceRepo device.Repository
}

func NewUpdateDeviceUseCase(deviceRepo device.Repository) *UpdateDeviceUseCase {
	return &UpdateDeviceUseCase{
		deviceRepo: deviceRepo,
	}
}

func (uc *UpdateDeviceUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[UpdateDeviceInput],
) (*UpdateDeviceOutput, error) {

	// [EN] The path device_id selects exactly which device row can be updated.
	// [中] path 的 device_id 決定本次只能更新哪一筆裝置資料。
	// [日] path の device_id により、この更新で対象となる端末行を特定する。
	deviceID, err := shared.ParseDeviceID(input.Data.DeviceID)
	if err != nil {
		return nil, ErrInvalidID
	}

	// [EN] Load first so invalid IDs and missing devices fail before any mutation.
	// [中] 先讀取資料，確保無效 ID 或不存在的裝置不會造成異動。
	// [日] 先に読み込み、無効な ID や存在しない端末で変更が発生しないようにする。
	deviceData, err := uc.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return nil, ErrDeviceNotFound
	}

	// [EN] Validate platform before assigning mutable fields.
	// [中] 先驗證 platform，再寫入可變欄位。
	// [日] 変更可能な項目へ代入する前に platform を検証する。
	platform, err := device.ParsePlatform(input.Data.Platform)
	if err != nil {
		return nil, ErrInvalidPlatform
	}

	deviceData.Name = input.Data.Name
	deviceData.Platform = platform

	if err := uc.deviceRepo.Update(ctx, deviceData); err != nil {
		return nil, ErrUpdateFailed
	}

	// [EN] Re-read after update so the response includes the database-owned updated_at timestamp.
	// [中] 更新後重新讀取，確保 response 帶出資料庫實際寫入的 updated_at。
	// [日] 更新後に再取得し、DB が管理する updated_at timestamp をレスポンスへ含める。
	updatedDevice, err := uc.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return nil, ErrUpdateFailed
	}

	return toUpdateDeviceOutput(updatedDevice), nil
}

func toUpdateDeviceOutput(d *device.Device) *UpdateDeviceOutput {
	return &UpdateDeviceOutput{
		DeviceID:  d.ID.String(),
		Name:      d.Name,
		Platform:  d.Platform.String(),
		CreatedAt: timeutil.Format(d.CreatedAt, timeutil.FormatISO),
		UpdatedAt: timeutil.Format(d.UpdatedAt, timeutil.FormatISO),
	}
}

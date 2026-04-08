package device

import (
	"context"
	"errors"
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type DeviceRepository struct {
	postgres.BaseRepo
}

var _ device.Repository = (*DeviceRepository)(nil)

func NewDeviceRepository(db *sqlx.DB) *DeviceRepository {
	return &DeviceRepository{
		BaseRepo: postgres.NewBaseRepo(db),
	}
}

func (r *DeviceRepository) FindByID(ctx context.Context, deviceID device.ID) (*device.Device, error) {
	query, args, err := Table.
		Select(Table.Columns...).
		Where(squirrel.Eq{"id": deviceID.String()}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build device query: %w", err)
	}

	rec, err := postgres.ScanOne[DeviceRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, device.ErrDeviceNotFound
		}
		return nil, fmt.Errorf("find device: %w", err)
	}

	return toDomain(rec)
}

func (r *DeviceRepository) FindAllByAccountID(ctx context.Context, accountID shared.AccountID) ([]*device.Device, error) {
	query, args, err := postgres.Builder.
		Select("d.id", "d.platform", "d.name", "d.created_at", "d.updated_at").
		From("public.devices d").
		Join("public.accounts_devices ad ON d.id = ad.device_id").
		Where(squirrel.Eq{"ad.account_id": int64(accountID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build devices query: %w", err)
	}

	records, err := postgres.ScanAll[DeviceRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("find devices: %w", err)
	}

	devices := make([]*device.Device, 0, len(records))
	for i := range records {
		d, err := toDomain(&records[i])
		if err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, nil
}

func (r *DeviceRepository) Create(ctx context.Context, d *device.Device) error {
	rec := toRecord(d)

	// [EN] Create reports ErrDeviceAlreadyExists so the usecase can run the update branch in the same transaction.
	// [中] Create 以 ErrDeviceAlreadyExists 回報衝突，讓 usecase 能在同一交易中走更新分支。
	// [日] Create は競合時に ErrDeviceAlreadyExists を返し、usecase が同じ transaction で更新分岐へ進めるようにする。
	query, args, err := Table.Insert().
		Columns("id", "platform", "name").
		Values(rec.ID, rec.Platform, rec.Name).
		Suffix("ON CONFLICT (id) DO NOTHING").
		ToSql()
	if err != nil {
		return err
	}

	result, err := r.GetDB(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("create device: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("create device rows affected: %w", err)
	}
	if rows == 0 {
		return device.ErrDeviceAlreadyExists
	}
	return nil
}

func (r *DeviceRepository) Update(ctx context.Context, d *device.Device) error {
	rec := toRecord(d)

	// [EN] Device updates only mutate editable fields and let PostgreSQL refresh updated_at.
	// [中] 裝置更新只修改可編輯欄位，並由 PostgreSQL 刷新 updated_at。
	// [日] 端末更新では編集可能な項目のみ変更し、updated_at は PostgreSQL に更新させる。
	query, args, err := Table.Update().
		Set("name", rec.Name).
		Set("platform", rec.Platform).
		Set("updated_at", squirrel.Expr("now()")).
		Where(squirrel.Eq{"id": rec.ID}).
		ToSql()
	if err != nil {
		return err
	}

	result, err := r.GetDB(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update device: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update device rows affected: %w", err)
	}
	if rows == 0 {
		return device.ErrDeviceNotFound
	}
	return nil
}

func (r *DeviceRepository) BindAccount(ctx context.Context, deviceID device.ID, accountID shared.AccountID) error {
	query, args, err := postgres.Builder.
		Insert("public.accounts_devices").
		Columns("device_id", "account_id").
		Values(deviceID.String(), int64(accountID)).
		Suffix("ON CONFLICT DO NOTHING").
		ToSql()
	if err != nil {
		return err
	}

	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

func (r *DeviceRepository) DeleteByAccount(ctx context.Context, deviceID device.ID, accountID shared.AccountID) error {
	query, args, err := postgres.Builder.
		Delete("public.devices").
		Where(
			squirrel.And{
				squirrel.Eq{"id": deviceID.String()},
				squirrel.Expr(
					"EXISTS (SELECT 1 FROM public.accounts_devices WHERE device_id = ? AND account_id = ?)",
					deviceID.String(), int64(accountID),
				),
			},
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete device query: %w", err)
	}

	result, err := r.GetDB(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete device rows affected: %w", err)
	}
	if rows == 0 {
		return device.ErrDeviceNotFound
	}
	return nil
}

var Table = postgres.Table{
	Name: "public.devices",
	Columns: []string{
		"id",
		"platform",
		"name",
		"created_at",
		"updated_at",
	},
}

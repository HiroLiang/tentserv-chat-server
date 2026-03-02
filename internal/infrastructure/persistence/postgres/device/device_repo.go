package device

import (
	"context"
	"errors"
	"fmt"

	"github.com/HiroLiang/goat-server/internal/domain/device"
	"github.com/HiroLiang/goat-server/internal/domain/user"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

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

type DeviceRepository struct {
	db *sqlx.DB
}

var _ device.Repository = (*DeviceRepository)(nil)

func NewDeviceRepository(db *sqlx.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) FindByID(ctx context.Context, deviceID device.ID) (*device.Device, error) {
	query, args, err := Table.
		Select(Table.Columns...).
		Where(squirrel.Eq{"id": string(deviceID)}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build device query: %w", err)
	}

	rec, err := postgres.ScanOne[DeviceRecord](ctx, r.db, query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, device.ErrDeviceNotFound
		}
		return nil, fmt.Errorf("find device: %w", err)
	}

	return toDomain(rec)
}

func (r *DeviceRepository) FindAllByUserID(ctx context.Context, userID user.ID) ([]*device.Device, error) {
	query, args, err := postgres.Builder.
		Select("d.id", "d.platform", "d.name", "d.created_at", "d.updated_at").
		From("public.devices d").
		Join("public.device_user du ON d.id = du.device_id").
		Where(squirrel.Eq{"du.user_id": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build devices query: %w", err)
	}

	records, err := postgres.ScanAll[DeviceRecord](ctx, r.db, query, args...)
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

	query, args, err := Table.Insert().
		Columns("id", "platform", "name").
		Values(rec.ID, rec.Platform, rec.Name).
		ToSql()
	if err != nil {
		return err
	}

	return postgres.Exec(ctx, r.db, query, args...)
}

func (r *DeviceRepository) Update(ctx context.Context, d *device.Device) error {
	rec := toRecord(d)

	query, args, err := Table.Update().
		Set("name", rec.Name).
		Set("platform", rec.Platform).
		Set("updated_at", squirrel.Expr("now()")).
		Where(squirrel.Eq{"id": rec.ID}).
		ToSql()
	if err != nil {
		return err
	}

	return postgres.Exec(ctx, r.db, query, args...)
}

func (r *DeviceRepository) BindUser(ctx context.Context, deviceID device.ID, userID user.ID) error {
	query, args, err := postgres.Builder.
		Insert("public.device_user").
		Columns("device_id", "user_id").
		Values(string(deviceID), userID).
		Suffix("ON CONFLICT DO NOTHING").
		ToSql()
	if err != nil {
		return err
	}

	return postgres.Exec(ctx, r.db, query, args...)
}

package e2ee

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userotpprekey"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var otpPreKeyTable = postgres.Table{
	Name: "public.user_one_time_prekeys",
	Columns: []string{
		"id", "user_id", "device_id", "key_id", "public_key", "uploaded_at",
	},
}

type OTPPreKeyRepository struct {
	postgres.BaseRepo
}

var _ userotpprekey.Repository = (*OTPPreKeyRepository)(nil)

func NewOTPPreKeyRepository(db *sqlx.DB) *OTPPreKeyRepository {
	return &OTPPreKeyRepository{BaseRepo: postgres.NewBaseRepo(db)}
}

// ConsumeOne atomically deletes and returns one OTP prekey.
// Uses FOR UPDATE SKIP LOCKED so concurrent callers each consume different keys.
func (r *OTPPreKeyRepository) ConsumeOne(
	ctx context.Context,
	userID user.ID,
	deviceID device.ID,
) (*userotpprekey.UserOTPPreKey, error) {
	const query = `
WITH victim AS (
    SELECT id FROM public.user_one_time_prekeys
    WHERE user_id = $1 AND device_id = $2
    LIMIT 1 FOR UPDATE SKIP LOCKED
)
DELETE FROM public.user_one_time_prekeys
USING victim
WHERE public.user_one_time_prekeys.id = victim.id
RETURNING id, user_id, device_id, key_id, public_key, uploaded_at`

	db := r.GetDB(ctx)
	rec, err := postgres.ScanOne[OTPPreKeyRecord](ctx, db, query, userID, deviceID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, userotpprekey.ErrPoolEmpty
		}
		return nil, fmt.Errorf("consume otp prekey: %w", err)
	}

	return toOTPPreKeyDomain(rec)
}

func (r *OTPPreKeyRepository) AddBatch(ctx context.Context, keys []*userotpprekey.UserOTPPreKey) error {
	if len(keys) == 0 {
		return nil
	}

	q := otpPreKeyTable.Insert().Columns("user_id", "device_id", "key_id", "public_key")
	for _, k := range keys {
		rec := toOTPPreKeyRecord(k)
		q = q.Values(rec.UserID, rec.DeviceID, rec.KeyID, rec.PublicKey)
	}

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("build insert otp prekeys: %w", err)
	}

	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

func (r *OTPPreKeyRepository) CountAvailable(
	ctx context.Context,
	userID user.ID,
	deviceID device.ID,
) (int, error) {
	query, args, err := postgres.Builder.
		Select("COUNT(*)").
		From(otpPreKeyTable.Name).
		Where(squirrel.Eq{"user_id": userID, "device_id": deviceID}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count otp prekeys: %w", err)
	}

	db := r.GetDB(ctx)
	var count int
	if err := db.QueryRowxContext(ctx, query, args...).Scan(&count); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("count otp prekeys: %w", err)
	}
	return count, nil
}

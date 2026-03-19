package e2ee

import (
	"context"
	"errors"
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/useridentitykey"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var identityKeyTable = postgres.Table{
	Name: "public.user_identity_keys",
	Columns: []string{
		"id", "user_id", "device_id", "public_key", "fingerprint", "uploaded_at",
	},
}

type IdentityKeyRepository struct {
	postgres.BaseRepo
}

var _ useridentitykey.Repository = (*IdentityKeyRepository)(nil)

func NewIdentityKeyRepository(db *sqlx.DB) *IdentityKeyRepository {
	return &IdentityKeyRepository{BaseRepo: postgres.NewBaseRepo(db)}
}

func (r *IdentityKeyRepository) FindByUserAndDevice(
	ctx context.Context,
	userID user.ID,
	deviceID device.ID,
) (*useridentitykey.UserIdentityKey, error) {
	query, args, err := identityKeyTable.Select(identityKeyTable.Columns...).
		Where(squirrel.Eq{"user_id": userID, "device_id": deviceID}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build identity key query: %w", err)
	}

	rec, err := postgres.ScanOne[IdentityKeyRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, useridentitykey.ErrNotFound
		}
		return nil, fmt.Errorf("find identity key: %w", err)
	}

	return toIdentityKeyDomain(rec)
}

func (r *IdentityKeyRepository) FindByUser(
	ctx context.Context,
	userID user.ID,
) ([]*useridentitykey.UserIdentityKey, error) {
	query, args, err := identityKeyTable.Select(identityKeyTable.Columns...).
		Where(squirrel.Eq{"user_id": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build identity keys query: %w", err)
	}

	records, err := postgres.ScanAll[IdentityKeyRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("scan identity keys: %w", err)
	}

	keys := make([]*useridentitykey.UserIdentityKey, 0, len(records))
	for _, rec := range records {
		k, err := toIdentityKeyDomain(&rec)
		if err != nil {
			return nil, fmt.Errorf("convert identity key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *IdentityKeyRepository) Upsert(ctx context.Context, key *useridentitykey.UserIdentityKey) error {
	rec := toIdentityKeyRecord(key)

	query, args, err := identityKeyTable.Insert().
		Columns("user_id", "device_id", "public_key", "fingerprint").
		Values(rec.UserID, rec.DeviceID, rec.PublicKey, rec.Fingerprint).
		Suffix(`ON CONFLICT (user_id, device_id) DO UPDATE
			SET public_key = EXCLUDED.public_key,
			    fingerprint = EXCLUDED.fingerprint,
			    uploaded_at = now()
			RETURNING id, uploaded_at`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build upsert identity key: %w", err)
	}

	db := r.GetDB(ctx)
	row := db.QueryRowxContext(ctx, query, args...)
	if err := row.Scan(&key.ID, &key.UploadedAt); err != nil {
		return fmt.Errorf("upsert identity key: %w", err)
	}
	return nil
}

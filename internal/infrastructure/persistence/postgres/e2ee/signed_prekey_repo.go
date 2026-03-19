package e2ee

import (
	"context"
	"errors"
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/usersignedprekey"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var signedPreKeyTable = postgres.Table{
	Name: "public.user_signed_prekeys",
	Columns: []string{
		"id", "user_id", "device_id", "key_id", "public_key", "signature",
		"is_active", "created_at", "expires_at",
	},
}

type SignedPreKeyRepository struct {
	postgres.BaseRepo
}

var _ usersignedprekey.Repository = (*SignedPreKeyRepository)(nil)

func NewSignedPreKeyRepository(db *sqlx.DB) *SignedPreKeyRepository {
	return &SignedPreKeyRepository{BaseRepo: postgres.NewBaseRepo(db)}
}

func (r *SignedPreKeyRepository) FindActive(
	ctx context.Context,
	userID user.ID,
	deviceID device.ID,
) (*usersignedprekey.UserSignedPreKey, error) {
	query, args, err := signedPreKeyTable.Select(signedPreKeyTable.Columns...).
		Where(squirrel.Eq{"user_id": userID, "device_id": deviceID, "is_active": true}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build signed prekey query: %w", err)
	}

	rec, err := postgres.ScanOne[SignedPreKeyRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, usersignedprekey.ErrNotFound
		}
		return nil, fmt.Errorf("find active signed prekey: %w", err)
	}

	return toSignedPreKeyDomain(rec)
}

func (r *SignedPreKeyRepository) FindByKeyID(
	ctx context.Context,
	userID user.ID,
	deviceID device.ID,
	keyID usersignedprekey.KeyID,
) (*usersignedprekey.UserSignedPreKey, error) {
	query, args, err := signedPreKeyTable.Select(signedPreKeyTable.Columns...).
		Where(squirrel.Eq{"user_id": userID, "device_id": deviceID, "key_id": keyID}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build signed prekey query: %w", err)
	}

	rec, err := postgres.ScanOne[SignedPreKeyRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, usersignedprekey.ErrNotFound
		}
		return nil, fmt.Errorf("find signed prekey by key_id: %w", err)
	}

	return toSignedPreKeyDomain(rec)
}

func (r *SignedPreKeyRepository) Add(ctx context.Context, key *usersignedprekey.UserSignedPreKey) error {
	rec := toSignedPreKeyRecord(key)

	query, args, err := signedPreKeyTable.Insert().
		Columns("user_id", "device_id", "key_id", "public_key", "signature", "is_active", "expires_at").
		Values(rec.UserID, rec.DeviceID, rec.KeyID, rec.PublicKey, rec.Signature, rec.IsActive, rec.ExpiresAt).
		Suffix("RETURNING id, created_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert signed prekey: %w", err)
	}

	db := r.GetDB(ctx)
	row := db.QueryRowxContext(ctx, query, args...)
	if err := row.Scan(&key.ID, &key.CreatedAt); err != nil {
		return fmt.Errorf("insert signed prekey: %w", err)
	}
	return nil
}

func (r *SignedPreKeyRepository) DeactivateAll(
	ctx context.Context,
	userID user.ID,
	deviceID device.ID,
) error {
	query, args, err := signedPreKeyTable.Update().
		Set("is_active", false).
		Where(squirrel.Eq{"user_id": userID, "device_id": deviceID, "is_active": true}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build deactivate signed prekeys: %w", err)
	}

	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

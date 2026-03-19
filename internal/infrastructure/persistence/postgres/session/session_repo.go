package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type SessionRepository struct {
	postgres.BaseRepo
}

func NewSessionRepository(db *sqlx.DB) *SessionRepository {
	return &SessionRepository{BaseRepo: postgres.NewBaseRepo(db)}
}

func (r *SessionRepository) Create(ctx context.Context, accountID int64, userID int64, deviceID string, refreshHash string, expiresAt time.Time) (int64, error) {
	query, args, err := Table.Insert().
		Columns("account_id", "user_id", "device_id", "refresh_token_hash", "expires_at").
		Values(accountID, userID, deviceID, refreshHash, expiresAt).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build create session query: %w", err)
	}

	var id int64
	if err := r.GetDB(ctx).QueryRowxContext(ctx, query, args...).Scan(&id); err != nil {
		return 0, fmt.Errorf("create session: %w", err)
	}
	return id, nil
}

func (r *SessionRepository) FindByID(ctx context.Context, id int64) (*SessionRecord, error) {
	query, args, err := Table.
		Select(Table.Columns...).
		Where(squirrel.Eq{"id": id}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find session query: %w", err)
	}

	rec, err := postgres.ScanOne[SessionRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, postgres.ErrNotFound
		}
		return nil, fmt.Errorf("find session by id: %w", err)
	}
	return rec, nil
}

func (r *SessionRepository) FindByRefreshTokenHash(ctx context.Context, hash string) (*SessionRecord, error) {
	query, args, err := Table.
		Select(Table.Columns...).
		Where(squirrel.Eq{"refresh_token_hash": hash}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find session query: %w", err)
	}

	rec, err := postgres.ScanOne[SessionRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, postgres.ErrNotFound
		}
		return nil, fmt.Errorf("find session by refresh token: %w", err)
	}
	return rec, nil
}

func (r *SessionRepository) FindActiveIDsByAccountID(ctx context.Context, accountID int64) ([]int64, error) {
	query, args, err := Table.
		Select("id").
		Where(squirrel.And{
			squirrel.Eq{"account_id": accountID},
			squirrel.Eq{"revoked": false},
			squirrel.Expr("expires_at > NOW()"),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find active sessions query: %w", err)
	}

	ids, err := postgres.ScanAll[int64](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("find active sessions: %w", err)
	}
	return ids, nil
}

func (r *SessionRepository) FindAllActiveIDs(ctx context.Context) ([]int64, error) {
	query, args, err := Table.
		Select("id").
		Where(squirrel.And{
			squirrel.Eq{"revoked": false},
			squirrel.Expr("expires_at > NOW()"),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find all active sessions query: %w", err)
	}

	ids, err := postgres.ScanAll[int64](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("find all active sessions: %w", err)
	}
	return ids, nil
}

func (r *SessionRepository) UpdateRefreshToken(ctx context.Context, id int64, newHash string, expiresAt time.Time) error {
	query, args, err := Table.Update().
		Set("refresh_token_hash", newHash).
		Set("expires_at", expiresAt).
		Set("last_used_at", squirrel.Expr("NOW()")).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update refresh token query: %w", err)
	}
	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

func (r *SessionRepository) UpdateExpiresAt(ctx context.Context, id int64, expiresAt time.Time) error {
	query, args, err := Table.Update().
		Set("expires_at", expiresAt).
		Set("last_used_at", squirrel.Expr("NOW()")).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update expires_at query: %w", err)
	}
	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

func (r *SessionRepository) UpdateLastUsedAt(ctx context.Context, id int64) error {
	query, args, err := Table.Update().
		Set("last_used_at", squirrel.Expr("NOW()")).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update last_used_at query: %w", err)
	}
	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

func (r *SessionRepository) Revoke(ctx context.Context, id int64) error {
	query, args, err := Table.Update().
		Set("revoked", true).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build revoke session query: %w", err)
	}
	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

func (r *SessionRepository) RevokeAllForAccount(ctx context.Context, accountID int64) error {
	query, args, err := Table.Update().
		Set("revoked", true).
		Where(squirrel.Eq{"account_id": accountID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build revoke all sessions query: %w", err)
	}
	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

func (r *SessionRepository) RevokeAll(ctx context.Context) error {
	query, args, err := Table.Update().
		Set("revoked", true).
		Where(squirrel.Eq{"revoked": false}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build revoke all sessions query: %w", err)
	}
	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

func (r *SessionRepository) UpdateUserID(ctx context.Context, id int64, userID int64) error {
	query, args, err := Table.Update().
		Set("user_id", userID).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update user_id query: %w", err)
	}
	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

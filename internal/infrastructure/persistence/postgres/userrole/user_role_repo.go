package userrole

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HiroLiang/goat-server/internal/domain/role"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
	"github.com/HiroLiang/goat-server/internal/domain/userrole"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres"
	dbRole "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/role"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var Table = postgres.Table{
	Name: "goat.public.users_roles",
	Columns: []string{
		"user_id",
		"role_id",
		"created_at",
	},
}

type UserRoleRepository struct {
	db *sqlx.DB
}

var _ userrole.Repository = (*UserRoleRepository)(nil)

func NewUserRoleRepository(db *sqlx.DB) *UserRoleRepository {
	return &UserRoleRepository{db: db}
}

func (r UserRoleRepository) FindRolesByUser(ctx context.Context, userID shared.UserID) ([]*role.Role, error) {
	query, args, err := postgres.Builder.
		Select("r.id", "r.type", "r.creator", "r.created_at", "r.updated_at").
		From(dbRole.Table.Name + " r").
		Join(Table.Name + " ur ON ur.role_id = r.id").
		Where(squirrel.Eq{"ur.user_id": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build user_role query: %w", err)
	}

	records, err := postgres.ScanAll[dbRole.RoleRecord](ctx, r.db, query, args...)
	if err != nil {
		return nil, fmt.Errorf("scan roles: %w", err)
	}

	roles := make([]*role.Role, 0, len(records))
	for _, rec := range records {
		r, err := dbRole.ToDomain(&rec)
		if err != nil {
			return nil, fmt.Errorf("convert role: %w", err)
		}
		roles = append(roles, r)
	}

	return roles, nil
}

func (r UserRoleRepository) Exists(ctx context.Context, userID shared.UserID, role role.Code) bool {
	query, args, err := Table.Select("1").
		From(Table.Name+" ur").
		LeftJoin(dbRole.Table.Name+" r ON ur.role_id = r.id").
		Where("ur.user_id = ? AND r.type = ?", userID, role).
		ToSql()
	if err != nil {
		return false
	}

	return postgres.Exists(ctx, r.db, query, args...)
}

func (r UserRoleRepository) Assign(ctx context.Context, userID shared.UserID, role role.Code) error {
	query, args, err := Table.Insert().
		Columns("user_id", "role_id").
		Select(
			postgres.Builder.
				Select().
				Column(squirrel.Expr("?", userID)).
				Column("id").
				From(dbRole.Table.Name).
				Where(squirrel.Eq{"type": role}),
		).
		Suffix("ON CONFLICT DO NOTHING RETURNING user_id").
		ToSql()
	if err != nil {
		return fmt.Errorf("build assign sql: %w", err)
	}

	err = r.db.QueryRowContext(ctx, query, args...).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return userrole.ErrUserRoleAlreadyAssigned
	}
	if err != nil {
		return err
	}

	return nil
}

func (r UserRoleRepository) Revoke(ctx context.Context, userID shared.UserID, role role.Code) error {
	query, args, err := Table.Delete().
		From(Table.Name + " ur").
		Where(squirrel.And{
			squirrel.Expr("ur.role_id = r.id"),
			squirrel.Eq{
				"ur.user_id": userID,
				"r.type":     role,
			},
		}).
		Suffix("USING " + dbRole.Table.Name + " r").
		ToSql()

	if err != nil {
		return fmt.Errorf("build revoke sql: %w", err)
	}

	err = postgres.Exec(ctx, r.db, query, args...)
	if err != nil {
		return userrole.ErrRevokeFailed
	}

	return nil
}

package userrole

import (
	"context"
	"fmt"

	"github.com/HiroLiang/goat-server/internal/domain/role"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
	"github.com/HiroLiang/goat-server/internal/domain/userrole"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres"
	postgresRole "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/role"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var JoinTable = postgres.Table{
	Name: "goat.public.users_roles",
	Columns: []string{
		"user_id",
		"role_id",
	},
}

var RolesTable = postgres.Table{
	Name:    "goat.public.roles",
	Columns: postgresRole.Table.Columns,
}

type UserRoleRepository struct {
	postgres.BaseRepo
}

var _ userrole.Repository = (*UserRoleRepository)(nil)

func NewUserRoleRepository(db *sqlx.DB) *UserRoleRepository {
	return &UserRoleRepository{
		BaseRepo: postgres.NewBaseRepo(db),
	}
}

func (r *UserRoleRepository) Assign(ctx context.Context, userID shared.UserID, roleCode role.Code) error {
	query := `INSERT INTO goat.public.users_roles (user_id, role_id)
SELECT $1, id FROM goat.public.roles WHERE code = $2
ON CONFLICT DO NOTHING`
	return postgres.Exec(ctx, r.GetDB(ctx), query, userID, roleCode)
}

func (r *UserRoleRepository) Revoke(ctx context.Context, userID shared.UserID, roleCode role.Code) error {
	query := `DELETE FROM goat.public.users_roles
WHERE user_id = $1 AND role_id = (SELECT id FROM goat.public.roles WHERE code = $2)`
	return postgres.Exec(ctx, r.GetDB(ctx), query, userID, roleCode)
}

func (r *UserRoleRepository) Exists(ctx context.Context, userID shared.UserID, roleCode role.Code) bool {
	query := `SELECT 1 FROM goat.public.users_roles ur
JOIN goat.public.roles ro ON ro.id = ur.role_id
WHERE ur.user_id = $1 AND ro.code = $2`
	return postgres.Exists(ctx, r.GetDB(ctx), query, userID, roleCode)
}

func (r *UserRoleRepository) FindRolesByUser(ctx context.Context, userID shared.UserID) ([]*role.Role, error) {
	query, args, err := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).
		Select(
			"ro.id", "ro.code", "ro.name", "ro.description",
			"ro.created_by", "ro.created_at", "ro.updated_at",
		).
		From("goat.public.users_roles ur").
		Join("goat.public.roles ro ON ro.id = ur.role_id").
		Where(squirrel.Eq{"ur.user_id": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find roles by user query: %w", err)
	}

	records, err := postgres.ScanAll[postgresRole.RoleRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("scan roles by user: %w", err)
	}

	roles := make([]*role.Role, 0, len(records))
	for i := range records {
		r, err := postgresRole.ToDomain(&records[i])
		if err != nil {
			return nil, fmt.Errorf("map role: %w", err)
		}
		roles = append(roles, r)
	}
	return roles, nil
}

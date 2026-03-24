package userrole

import (
	"context"
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userrole"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	postgresRole "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/role"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var JoinTable = postgres.Table{
	Name: "public.users_roles",
	Columns: []string{
		"user_id",
		"role_id",
	},
}

var RolesTable = postgres.Table{
	Name:    "public.roles",
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
	query := `INSERT INTO public.users_roles (user_id, role_id)
SELECT $1, id FROM public.roles WHERE code = $2
ON CONFLICT DO NOTHING`
	return postgres.Exec(ctx, r.GetDB(ctx), query, userID, roleCode)
}

func (r *UserRoleRepository) Revoke(ctx context.Context, userID shared.UserID, roleCode role.Code) error {
	query := `DELETE FROM public.users_roles
WHERE user_id = $1 AND role_id = (SELECT id FROM public.roles WHERE code = $2)`
	return postgres.Exec(ctx, r.GetDB(ctx), query, userID, roleCode)
}

func (r *UserRoleRepository) Exists(ctx context.Context, userID shared.UserID, roleCode role.Code) bool {
	query := `SELECT 1 FROM public.users_roles ur
JOIN public.roles ro ON ro.id = ur.role_id
WHERE ur.user_id = $1 AND ro.code = $2`
	return postgres.Exists(ctx, r.GetDB(ctx), query, userID, roleCode)
}

func (r *UserRoleRepository) FindRolesByUser(ctx context.Context, userID shared.UserID) ([]*role.Role, error) {
	query, args, err := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).
		Select(
			"ro.id", "ro.code", "ro.name", "ro.description",
			"ro.created_by", "ro.created_at", "ro.updated_at",
		).
		From("public.users_roles ur").
		Join("public.roles ro ON ro.id = ur.role_id").
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

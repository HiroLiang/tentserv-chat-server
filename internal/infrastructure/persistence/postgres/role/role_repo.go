package role

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var Table = postgres.Table{
	Name: "goat.public.roles",
	Columns: []string{
		"id",
		"code",
		"name",
		"description",
		"created_by",
		"created_at",
		"updated_at",
	},
}

type RoleRepository struct {
	db *sqlx.DB
}

var _ role.Repository = (*RoleRepository)(nil)

func NewRoleRepository(db *sqlx.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) FindByID(ctx context.Context, id shared.RoleID) (*role.Role, error) {
	return r.findOneBy(ctx, squirrel.Eq{"id": id})
}

func (r *RoleRepository) FindByCode(ctx context.Context, code role.Code) (*role.Role, error) {
	return r.findOneBy(ctx, squirrel.Eq{"code": code})
}

func (r *RoleRepository) FindAll(ctx context.Context) ([]*role.Role, error) {
	query, args, err := Table.Select(Table.Columns...).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build role list query: %w", err)
	}

	records, err := postgres.ScanAll[RoleRecord](ctx, r.db, query, args...)
	if err != nil {
		return nil, fmt.Errorf("scan roles: %w", err)
	}

	roles := make([]*role.Role, 0, len(records))
	for i := range records {
		domain, mapErr := ToDomain(&records[i])
		if mapErr != nil {
			return nil, fmt.Errorf("map role: %w", mapErr)
		}
		roles = append(roles, domain)
	}
	return roles, nil
}

func (r *RoleRepository) Create(ctx context.Context, roleDomain *role.Role) error {
	rec := ToRecord(roleDomain)
	query, args, err := Table.Insert().
		Columns("code", "name", "description", "created_by").
		Values(rec.Code, rec.Name, rec.Description, rec.CreatedBy).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert role query: %w", err)
	}

	if err := postgres.Exec(ctx, r.db, query, args...); err != nil {
		if isUniqueViolation(err) {
			return role.ErrAlreadyExists
		}
		return fmt.Errorf("insert role: %w", err)
	}
	return nil
}

func (r *RoleRepository) Update(ctx context.Context, roleDomain *role.Role) error {
	rec := ToRecord(roleDomain)
	query, args, err := Table.Update().
		Set("code", rec.Code).
		Set("name", rec.Name).
		Set("description", rec.Description).
		Set("created_by", rec.CreatedBy).
		Set("updated_at", squirrel.Expr("now()")).
		Where(squirrel.Eq{"id": rec.ID}).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("build update role query: %w", err)
	}

	var updatedID shared.RoleID
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&updatedID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return role.ErrNotFound
		}
		if isUniqueViolation(err) {
			return role.ErrAlreadyExists
		}
		return fmt.Errorf("update role: %w", err)
	}
	return nil
}

func (r *RoleRepository) findOneBy(ctx context.Context, cond squirrel.Sqlizer) (*role.Role, error) {
	query, args, err := Table.Select(Table.Columns...).
		Where(cond).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build role query: %w", err)
	}

	record, err := postgres.ScanOne[RoleRecord](ctx, r.db, query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, role.ErrNotFound
		}
		return nil, fmt.Errorf("find role: %w", err)
	}
	return ToDomain(record)
}

func isUniqueViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key value")
}

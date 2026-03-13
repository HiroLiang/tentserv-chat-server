package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/HiroLiang/goat-server/internal/domain/shared"
	"github.com/HiroLiang/goat-server/internal/domain/user"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres"
	dbRole "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/role"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var Table = postgres.Table{
	Name: "public.users",
	Columns: []string{
		"id",
		"name",
		"email",
		"password",
		"user_status",
		"user_ip",
		"avatar_name",
		"created_at",
		"updated_at",
	},
}

type UserRepository struct {
	db *sqlx.DB
}

func (r *UserRepository) FindByAccountID(ctx context.Context, accountID shared.AccountID) (*[]user.User, error) {
	//TODO implement me
	panic("implement me")
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *user.User) (shared.UserID, error) {
	//TODO implement me
	panic("implement me")
}

func (r *UserRepository) FindByID(ctx context.Context, id shared.UserID) (*user.User, error) {
	return r.findOneBy(ctx, squirrel.Eq{"id": id})
}

func (r *UserRepository) FindByEmail(ctx context.Context, email shared.EmailAddress) (*user.User, error) {
	return r.findOneBy(ctx, squirrel.Eq{"email": email})
}

func (r *UserRepository) CreateWithRole(ctx context.Context, u *user.User, roleType string) error {
	record := toRecord(u)

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	insertUserQuery, insertUserArgs, err := Table.Insert().
		Columns("name").
		Values(record.Name).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert user query: %w", err)
	}

	var userID shared.UserID
	if err := tx.QueryRowContext(ctx, insertUserQuery, insertUserArgs...).Scan(&userID); err != nil {
		if isUniqueViolation(err) {
			return user.ErrUserAlreadyExists
		}
		return fmt.Errorf("insert user: %w", err)
	}

	roleIDQuery, roleIDArgs, err := postgres.Builder.
		Select("id").
		From(dbRole.Table.Name).
		Where(squirrel.Eq{"type": roleType}).
		Limit(1).
		ToSql()
	if err != nil {
		return fmt.Errorf("build role lookup query: %w", err)
	}

	var roleID int64
	if err := tx.QueryRowContext(ctx, roleIDQuery, roleIDArgs...).Scan(&roleID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.ErrDefaultRoleNotFound
		}
		return fmt.Errorf("find default role: %w", err)
	}

	insertUserRoleQuery, insertUserRoleArgs, err := postgres.Builder.
		Insert("goat.public.users_roles").
		Columns("user_id", "role_id").
		Values(userID, roleID).
		ToSql()
	if err != nil {
		return fmt.Errorf("build user role query: %w", err)
	}

	if _, err := tx.ExecContext(ctx, insertUserRoleQuery, insertUserRoleArgs...); err != nil {
		return fmt.Errorf("insert user role: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	committed = true
	u.ID = userID
	return nil
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	rec := toRecord(u)

	query, args, err := Table.Update().
		Set("name", rec.Name).
		Set("avatar_name", rec.AvatarName).
		Where(squirrel.Eq{"id": rec.ID}).
		ToSql()
	if err != nil {
		return err
	}

	return postgres.Exec(ctx, r.db, query, args...)
}

func (r *UserRepository) findOneBy(
	ctx context.Context,
	cond squirrel.Eq,
) (*user.User, error) {

	query, args, err := Table.
		Select(Table.Columns...).
		Where(cond).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build user query: %w", err)
	}

	rec, err := postgres.ScanOne[UserRecord](ctx, r.db, query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	return toDomain(rec)
}

func isUniqueViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key value")
}

var _ user.Repository = (*UserRepository)(nil)

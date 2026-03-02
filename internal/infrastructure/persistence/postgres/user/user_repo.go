package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/HiroLiang/goat-server/internal/domain/user"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres"
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

var _ user.Repository = (*UserRepository)(nil)

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(ctx context.Context, id user.ID) (*user.User, error) {
	return r.findOneBy(ctx, squirrel.Eq{"id": id})
}

func (r *UserRepository) FindByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	return r.findOneBy(ctx, squirrel.Eq{"email": email})
}

func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	record := toRecord(u)

	query, args, err := Table.Insert().
		Columns("name", "email", "password", "user_status", "user_ip").
		Values(record.Name, record.Email, record.Password, record.UserStatus, record.UserIP).
		ToSql()
	if err != nil {
		return err
	}

	return postgres.Exec(ctx, r.db, query, args...)
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	rec := toRecord(u)

	query, args, err := Table.Update().
		Set("name", rec.Name).
		Set("password", rec.Password).
		Set("user_status", rec.UserStatus).
		Set("user_ip", rec.UserIP).
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

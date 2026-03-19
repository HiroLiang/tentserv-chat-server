package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	postgres.BaseRepo
}

var _ user.Repository = (*UserRepository)(nil)

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{
		BaseRepo: postgres.NewBaseRepo(db),
	}
}

func (r *UserRepository) FindByID(ctx context.Context, id shared.UserID) (*user.User, error) {
	return r.findOneBy(ctx, squirrel.Eq{"id": id})
}

func (r *UserRepository) Create(ctx context.Context, u *user.User) (shared.UserID, error) {
	rec := toRecord(u)

	query, args, err := Table.Insert().
		Columns("account_id", "name").
		Values(rec.AccountID, rec.Name).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build insert user query: %w", err)
	}

	var userID shared.UserID
	if err := r.GetDB(ctx).QueryRowxContext(ctx, query, args...).Scan(&userID); err != nil {
		if isUniqueViolation(err) {
			return 0, user.ErrUserAlreadyExists
		}
		return 0, fmt.Errorf("insert user: %w", err)
	}

	u.ID = userID
	return userID, nil
}

func (r *UserRepository) FindByAccountID(ctx context.Context, accountID shared.AccountID) (*[]user.User, error) {
	query, args, err := Table.
		Select(Table.Columns...).
		Where(squirrel.Eq{"account_id": accountID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build users query: %w", err)
	}

	records, err := postgres.ScanAll[UserRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("scan users: %w", err)
	}

	users := make([]user.User, 0, len(records))
	for i := range records {
		u, err := toDomain(&records[i])
		if err != nil {
			return nil, fmt.Errorf("convert user: %w", err)
		}
		users = append(users, *u)
	}
	return &users, nil
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	rec := toRecord(u)

	query, args, err := Table.Update().
		Set("name", rec.Name).
		Set("avatar", rec.AvatarName).
		Where(squirrel.Eq{"id": rec.ID}).
		ToSql()
	if err != nil {
		return err
	}

	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
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

	rec, err := postgres.ScanOne[UserRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("find user: %d", err))
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	userData, err := toDomain(rec)
	if err != nil {
		return nil, fmt.Errorf("convert user: %w", err)
	}

	roleCodes, err := r.findRoleCodes(ctx, userData.ID)
	if err != nil {
		return nil, fmt.Errorf("find user roles: %w", err)
	}
	userData.RoleCodes = roleCodes

	return userData, nil
}

func (r *UserRepository) findRoleCodes(ctx context.Context, id shared.UserID) ([]role.Code, error) {
	query, args, err := JoinTable.
		Select("roles.code").
		Join("roles ON roles.id = users_roles.role_id").
		Where(squirrel.Eq{"user_id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build user roles query: %w", err)
	}

	codes, err := postgres.ScanAll[role.Code](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("scan user roles: %w", err)
	}

	return codes, nil

}

func isUniqueViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key value")
}

var Table = postgres.Table{
	Name: "goat.public.users",
	Columns: []string{
		"id",
		"account_id",
		"name",
		"avatar",
		"created_at",
		"updated_at",
	},
}

var JoinTable = postgres.Table{
	Name: "goat.public.users_roles",
	Columns: []string{
		"user_id",
		"role_id",
	},
}

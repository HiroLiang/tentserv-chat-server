package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
)

type UserSearchRecord struct {
	ID          shared.UserID    `db:"id"`
	Name        string           `db:"name"`
	AvatarName  *string          `db:"avatar"`
	AccountID   shared.AccountID `db:"account_id"`
	PublicID    uuid.UUID        `db:"public_id"`
	AccountName string           `db:"account"`
	CreatedAt   time.Time        `db:"created_at"`
	UpdatedAt   time.Time        `db:"updated_at"`
}

func toSearchDomain(r *UserSearchRecord) *user.UserSearchResult {
	avatar := ""
	if r.AvatarName != nil {
		avatar = *r.AvatarName
	}
	return &user.UserSearchResult{
		ID:          r.ID,
		Name:        r.Name,
		Avatar:      avatar,
		PublicID:    r.PublicID.String(),
		AccountName: r.AccountName,
	}
}

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

var searchColumns = []string{
	"u.id", "u.name", "u.avatar", "u.account_id", "u.created_at", "u.updated_at", "a.public_id", "a.account",
}

func (r *UserRepository) searchJoinQuery(ctx context.Context, cond squirrel.Sqlizer) ([]*user.UserSearchResult, error) {
	query, args, err := squirrel.Select(searchColumns...).
		From("goat.public.users u").
		Join("goat.public.accounts a ON u.account_id = a.id").
		Where(cond).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build search query: %w", err)
	}

	records, err := postgres.ScanAll[UserSearchRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("scan search results: %w", err)
	}

	results := make([]*user.UserSearchResult, 0, len(records))
	for i := range records {
		results = append(results, toSearchDomain(&records[i]))
	}
	return results, nil
}

func (r *UserRepository) SearchByName(ctx context.Context, keyword string) ([]*user.UserSearchResult, error) {
	return r.searchJoinQuery(ctx, squirrel.ILike{"u.name": "%" + keyword + "%"})
}

func (r *UserRepository) FindByAccountName(ctx context.Context, accountName string) ([]*user.UserSearchResult, error) {
	return r.searchJoinQuery(ctx, squirrel.Eq{"a.account": accountName})
}

func (r *UserRepository) FindByPublicID(ctx context.Context, publicID string) ([]*user.UserSearchResult, error) {
	return r.searchJoinQuery(ctx, squirrel.Eq{"a.public_id": publicID})
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

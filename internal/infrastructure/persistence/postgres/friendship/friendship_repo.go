package friendship

import (
	"context"
	"errors"
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type FriendshipRepository struct {
	postgres.BaseRepo
}

var _ friendship.Repository = (*FriendshipRepository)(nil)

var Table = postgres.Table{
	Name: "goat.public.user_friendships",
	Columns: []string{
		"id",
		"user_id",
		"friend_id",
		"status",
		"created_at",
		"updated_at",
	},
}

func NewFriendshipRepository(db *sqlx.DB) *FriendshipRepository {
	return &FriendshipRepository{
		BaseRepo: postgres.NewBaseRepo(db),
	}
}

func (r *FriendshipRepository) FindByUserID(ctx context.Context, userID shared.UserID) ([]*friendship.Friendship, error) {
	query, args, err := Table.
		Select(Table.Columns...).
		Where(squirrel.Eq{"user_id": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build friendship query: %w", err)
	}

	records, err := postgres.ScanAll[FriendshipRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("scan friendships: %w", err)
	}

	result := make([]*friendship.Friendship, 0, len(records))
	for i := range records {
		result = append(result, toDomain(&records[i]))
	}
	return result, nil
}

func (r *FriendshipRepository) Create(ctx context.Context, userID, friendID shared.UserID) error {
	query, args, err := squirrel.
		Insert(Table.Name).
		Columns("user_id", "friend_id", "status").
		Values(userID, friendID, string(friendship.StatusPending)).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build create friendship query: %w", err)
	}
	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

func (r *FriendshipRepository) FindByID(ctx context.Context, id int64) (*friendship.Friendship, error) {
	query, args, err := Table.
		Select(Table.Columns...).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find friendship by id query: %w", err)
	}

	rec, err := postgres.ScanOne[FriendshipRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, friendship.ErrFriendshipNotFound
		}
		return nil, fmt.Errorf("scan friendship: %w", err)
	}
	return toDomain(rec), nil
}

func (r *FriendshipRepository) FindByUserIDAndFriendID(ctx context.Context, userID, friendID shared.UserID) (*friendship.Friendship, error) {
	query, args, err := Table.
		Select(Table.Columns...).
		Where(squirrel.Eq{"user_id": userID, "friend_id": friendID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find friendship by user and friend query: %w", err)
	}

	rec, err := postgres.ScanOne[FriendshipRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, friendship.ErrFriendshipNotFound
		}
		return nil, fmt.Errorf("scan friendship: %w", err)
	}
	return toDomain(rec), nil
}

func (r *FriendshipRepository) FindBetweenUsers(ctx context.Context, userID1, userID2 shared.UserID) (*friendship.Friendship, error) {
	query, args, err := Table.
		Select(Table.Columns...).
		Where(squirrel.Or{
			squirrel.And{squirrel.Eq{"user_id": userID1}, squirrel.Eq{"friend_id": userID2}},
			squirrel.And{squirrel.Eq{"user_id": userID2}, squirrel.Eq{"friend_id": userID1}},
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find friendship between users query: %w", err)
	}

	rec, err := postgres.ScanOne[FriendshipRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, friendship.ErrFriendshipNotFound
		}
		return nil, fmt.Errorf("scan friendship: %w", err)
	}
	return toDomain(rec), nil
}

func (r *FriendshipRepository) FindPendingByFriendID(ctx context.Context, friendID shared.UserID) ([]*friendship.Friendship, error) {
	query, args, err := Table.
		Select(Table.Columns...).
		Where(squirrel.Eq{
			"friend_id": friendID,
			"status":    string(friendship.StatusPending),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find pending friendships query: %w", err)
	}

	records, err := postgres.ScanAll[FriendshipRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("scan pending friendships: %w", err)
	}

	result := make([]*friendship.Friendship, 0, len(records))
	for i := range records {
		result = append(result, toDomain(&records[i]))
	}
	return result, nil
}

func (r *FriendshipRepository) Delete(ctx context.Context, id int64) error {
	query, args, err := squirrel.
		Delete(Table.Name).
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete friendship query: %w", err)
	}
	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

func (r *FriendshipRepository) UpdateStatus(ctx context.Context, id int64, status friendship.Status) error {
	query, args, err := squirrel.
		Update(Table.Name).
		Set("status", string(status)).
		Set("updated_at", squirrel.Expr("now()")).
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update friendship status query: %w", err)
	}
	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

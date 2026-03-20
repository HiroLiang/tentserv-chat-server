package friendship

import (
	"context"
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
		Where(squirrel.Eq{
			"user_id": userID,
			"status":  string(friendship.StatusAccepted),
		}).
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

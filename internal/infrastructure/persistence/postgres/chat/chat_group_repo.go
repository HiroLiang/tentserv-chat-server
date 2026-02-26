package chat

import (
	"context"
	"errors"
	"fmt"

	"github.com/HiroLiang/goat-server/internal/domain/chatgroup"
	"github.com/HiroLiang/goat-server/internal/domain/participant"
	"github.com/HiroLiang/goat-server/internal/domain/user"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var ChatGroupTable = postgres.Table{
	Name: "public.chat_groups",
	Columns: []string{
		"id",
		"name",
		"description",
		"avatar_url",
		"type",
		"max_members",
		"is_deleted",
		"created_at",
		"updated_at",
		"created_by",
	},
}

type ChatGroupRepository struct {
	db *sqlx.DB
}

var _ chatgroup.Repository = (*ChatGroupRepository)(nil)

func NewChatGroupRepository(db *sqlx.DB) *ChatGroupRepository {
	return &ChatGroupRepository{db: db}
}

func (r *ChatGroupRepository) FindByID(ctx context.Context, id chatgroup.ID) (*chatgroup.ChatGroup, error) {
	query, args, err := ChatGroupTable.Select(ChatGroupTable.Columns...).
		Where(squirrel.Eq{"id": id}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build chat group query: %w", err)
	}

	rec, err := postgres.ScanOne[ChatGroupRecord](ctx, r.db, query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, chatgroup.ErrNotFound
		}
		return nil, fmt.Errorf("find chat group: %w", err)
	}

	return toChatGroupDomain(rec)
}

func (r *ChatGroupRepository) FindByCreator(ctx context.Context, creatorID user.ID) ([]*chatgroup.ChatGroup, error) {
	query, args, err := ChatGroupTable.Select(ChatGroupTable.Columns...).
		Where(squirrel.Eq{"created_by": creatorID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build chat groups query: %w", err)
	}

	records, err := postgres.ScanAll[ChatGroupRecord](ctx, r.db, query, args...)
	if err != nil {
		return nil, fmt.Errorf("scan chat groups: %w", err)
	}

	groups := make([]*chatgroup.ChatGroup, 0, len(records))
	for _, rec := range records {
		g, err := toChatGroupDomain(&rec)
		if err != nil {
			return nil, fmt.Errorf("convert chat group: %w", err)
		}
		groups = append(groups, g)
	}

	return groups, nil
}

func (r *ChatGroupRepository) Create(ctx context.Context, g *chatgroup.ChatGroup) error {
	rec := toChatGroupRecord(g)

	query, args, err := ChatGroupTable.Insert().
		Columns("name", "description", "avatar_url", "type", "max_members", "created_by").
		Values(rec.Name, rec.Description, rec.AvatarURL, rec.Type, rec.MaxMembers, rec.CreatedBy).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert chat group: %w", err)
	}

	var id chatgroup.ID
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return fmt.Errorf("insert chat group: %w", err)
	}
	g.ID = id
	return nil
}

func (r *ChatGroupRepository) FindDirectByParticipants(
	ctx context.Context,
	p1ID, p2ID participant.ID,
) (*chatgroup.ChatGroup, error) {
	query := `
		SELECT cg.id, cg.name, cg.description, cg.avatar_url, cg.type, cg.max_members,
		       cg.is_deleted, cg.created_at, cg.updated_at, cg.created_by
		FROM public.chat_groups cg
		JOIN public.chat_group_members m1 ON cg.id = m1.group_id AND m1.participant_id = $1
		JOIN public.chat_group_members m2 ON cg.id = m2.group_id AND m2.participant_id = $2
		WHERE cg.type = 'direct' AND NOT cg.is_deleted
		LIMIT 1`

	rec, err := postgres.ScanOne[ChatGroupRecord](ctx, r.db, query, p1ID, p2ID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, chatgroup.ErrNotFound
		}
		return nil, fmt.Errorf("find direct group by participants: %w", err)
	}

	return toChatGroupDomain(rec)
}

func (r *ChatGroupRepository) Update(ctx context.Context, g *chatgroup.ChatGroup) error {
	rec := toChatGroupRecord(g)

	query, args, err := ChatGroupTable.Update().
		Set("name", rec.Name).
		Set("description", rec.Description).
		Set("avatar_url", rec.AvatarURL).
		Set("max_members", rec.MaxMembers).
		Set("updated_at", squirrel.Expr("now()")).
		Where(squirrel.Eq{"id": rec.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update chat group: %w", err)
	}

	return postgres.Exec(ctx, r.db, query, args...)
}

func (r *ChatGroupRepository) SoftDelete(ctx context.Context, id chatgroup.ID) error {
	query, args, err := ChatGroupTable.Update().
		Set("is_deleted", true).
		Set("updated_at", squirrel.Expr("now()")).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build soft delete chat group: %w", err)
	}

	return postgres.Exec(ctx, r.db, query, args...)
}

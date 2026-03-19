package chat

import (
	"context"
	"errors"
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var ChatRoomTable = postgres.Table{
	Name: "public.chat_rooms",
	Columns: []string{
		"id",
		"name",
		"description",
		"avatar_name",
		"type",
		"max_members",
		"allow_agent",
		"is_deleted",
		"created_at",
		"updated_at",
	},
}

type ChatRoomRepository struct {
	db *sqlx.DB
}

var _ chatroom.Repository = (*ChatRoomRepository)(nil)

func NewChatRoomRepository(db *sqlx.DB) *ChatRoomRepository {
	return &ChatRoomRepository{db: db}
}

func (r *ChatRoomRepository) FindByID(ctx context.Context, id chatroom.ID) (*chatroom.ChatRoom, error) {
	query, args, err := ChatRoomTable.Select(ChatRoomTable.Columns...).
		Where(squirrel.Eq{"id": id}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build chat room query: %w", err)
	}

	rec, err := postgres.ScanOne[ChatRoomRecord](ctx, r.db, query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, chatroom.ErrNotFound
		}
		return nil, fmt.Errorf("find chat room: %w", err)
	}

	return toChatRoomDomain(rec)
}

func (r *ChatRoomRepository) Create(ctx context.Context, room *chatroom.ChatRoom) error {
	rec := toChatRoomRecord(room)

	query, args, err := ChatRoomTable.Insert().
		Columns("name", "description", "avatar_name", "type", "max_members", "allow_agent").
		Values(rec.Name, rec.Description, rec.AvatarName, rec.Type, rec.MaxMembers, rec.AllowAgent).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert chat room: %w", err)
	}

	var id chatroom.ID
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return fmt.Errorf("insert chat room: %w", err)
	}
	room.ID = id
	return nil
}

func (r *ChatRoomRepository) FindDirectByParticipants(
	ctx context.Context,
	p1ID, p2ID participant.ID,
) (*chatroom.ChatRoom, error) {
	query := `
		SELECT cr.id, cr.name, cr.description, cr.avatar_name, cr.type, cr.max_members,
		       cr.allow_agent, cr.is_deleted, cr.created_at, cr.updated_at
		FROM public.chat_rooms cr
		JOIN public.chat_members m1 ON cr.id = m1.room_id AND m1.participant_id = $1
		JOIN public.chat_members m2 ON cr.id = m2.room_id AND m2.participant_id = $2
		WHERE cr.type = 'direct' AND NOT cr.is_deleted
		LIMIT 1`

	rec, err := postgres.ScanOne[ChatRoomRecord](ctx, r.db, query, p1ID, p2ID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, chatroom.ErrNotFound
		}
		return nil, fmt.Errorf("find direct room by participants: %w", err)
	}

	return toChatRoomDomain(rec)
}

func (r *ChatRoomRepository) Update(ctx context.Context, room *chatroom.ChatRoom) error {
	rec := toChatRoomRecord(room)

	query, args, err := ChatRoomTable.Update().
		Set("name", rec.Name).
		Set("description", rec.Description).
		Set("avatar_name", rec.AvatarName).
		Set("max_members", rec.MaxMembers).
		Set("allow_agent", rec.AllowAgent).
		Set("updated_at", squirrel.Expr("now()")).
		Where(squirrel.Eq{"id": rec.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update chat room: %w", err)
	}

	return postgres.Exec(ctx, r.db, query, args...)
}

func (r *ChatRoomRepository) SoftDelete(ctx context.Context, id chatroom.ID) error {
	query, args, err := ChatRoomTable.Update().
		Set("is_deleted", true).
		Set("updated_at", squirrel.Expr("now()")).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build soft delete chat room: %w", err)
	}

	return postgres.Exec(ctx, r.db, query, args...)
}

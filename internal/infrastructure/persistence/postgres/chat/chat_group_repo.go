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
	postgres.BaseRepo
}

var _ chatroom.Repository = (*ChatRoomRepository)(nil)

func NewChatRoomRepository(db *sqlx.DB) *ChatRoomRepository {
	return &ChatRoomRepository{BaseRepo: postgres.NewBaseRepo(db)}
}

func (r *ChatRoomRepository) FindByID(ctx context.Context, id chatroom.ID) (*chatroom.ChatRoom, error) {
	query, args, err := ChatRoomTable.Select(ChatRoomTable.Columns...).
		Where(squirrel.Eq{"id": id}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build chat room query: %w", err)
	}

	rec, err := postgres.ScanOne[ChatRoomRecord](ctx, r.GetDB(ctx), query, args...)
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
	if err := r.GetDB(ctx).QueryRowxContext(ctx, query, args...).Scan(&id); err != nil {
		return fmt.Errorf("insert chat room: %w", err)
	}
	room.ID = id
	return nil
}

// FindDirectByParticipants finds an active direct room shared by both participants.
// Only considers members where is_deleted = false to avoid returning rooms where
// a participant has been soft-deleted (C-3).
func (r *ChatRoomRepository) FindDirectByParticipants(
	ctx context.Context,
	p1ID, p2ID participant.ID,
) (*chatroom.ChatRoom, error) {
	query := `
		SELECT cr.id, cr.name, cr.description, cr.avatar_name, cr.type, cr.max_members,
		       cr.allow_agent, cr.is_deleted, cr.created_at, cr.updated_at
		FROM public.chat_rooms cr
		JOIN public.chat_members m1 ON cr.id = m1.room_id AND m1.participant_id = $1 AND m1.is_deleted = false
		JOIN public.chat_members m2 ON cr.id = m2.room_id AND m2.participant_id = $2 AND m2.is_deleted = false
		WHERE cr.type = 'direct' AND NOT cr.is_deleted
		LIMIT 1`

	rec, err := postgres.ScanOne[ChatRoomRecord](ctx, r.GetDB(ctx), query, p1ID, p2ID)
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

	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
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

	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

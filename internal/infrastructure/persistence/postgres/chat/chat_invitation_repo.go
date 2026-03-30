package chat

import (
	"context"
	"errors"
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatinvitation"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var ChatInvitationTable = postgres.Table{
	Name: "public.chat_invitations",
	Columns: []string{
		"id",
		"room_id",
		"inviter_id",
		"invitee_id",
		"status",
		"invitation_type",
		"expires_at",
		"created_at",
		"updated_at",
	},
}

// notExpired is the squirrel condition that filters out expired invitations.
var notExpired = squirrel.Or{
	squirrel.Eq{"expires_at": nil},
	squirrel.Gt{"expires_at": squirrel.Expr("now()")},
}

type ChatInvitationRepository struct {
	postgres.BaseRepo
}

var _ chatinvitation.Repository = (*ChatInvitationRepository)(nil)

func NewChatInvitationRepository(db *sqlx.DB) *ChatInvitationRepository {
	return &ChatInvitationRepository{BaseRepo: postgres.NewBaseRepo(db)}
}

func (r *ChatInvitationRepository) Create(ctx context.Context, inv *chatinvitation.ChatInvitation) error {
	rec := toInvitationRecord(inv)

	query, args, err := ChatInvitationTable.Insert().
		Columns("room_id", "inviter_id", "invitee_id", "status", "invitation_type", "expires_at").
		Values(rec.RoomID, rec.InviterID, rec.InviteeID, rec.Status, rec.InvitationType, rec.ExpiresAt).
		Suffix("RETURNING id, created_at, updated_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert chat invitation: %w", err)
	}

	if err := r.GetDB(ctx).QueryRowxContext(ctx, query, args...).Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
		return fmt.Errorf("insert chat invitation: %w", err)
	}
	return nil
}

func (r *ChatInvitationRepository) FindByID(ctx context.Context, id chatinvitation.ID) (*chatinvitation.ChatInvitation, error) {
	query, args, err := ChatInvitationTable.Select(ChatInvitationTable.Columns...).
		Where(squirrel.Eq{"id": id}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build chat invitation query: %w", err)
	}

	rec, err := postgres.ScanOne[ChatInvitationRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, chatinvitation.ErrNotFound
		}
		return nil, fmt.Errorf("find chat invitation: %w", err)
	}

	return toInvitationDomain(rec), nil
}

func (r *ChatInvitationRepository) FindByRoomAndInvitee(
	ctx context.Context,
	roomID chatroom.ID,
	inviteeID participant.ID,
) (*chatinvitation.ChatInvitation, error) {
	query, args, err := ChatInvitationTable.Select(ChatInvitationTable.Columns...).
		Where(squirrel.And{
			squirrel.Eq{"room_id": roomID, "invitee_id": inviteeID, "status": chatinvitation.Pending},
			notExpired,
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build chat invitation query: %w", err)
	}

	rec, err := postgres.ScanOne[ChatInvitationRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, chatinvitation.ErrNotFound
		}
		return nil, fmt.Errorf("find chat invitation by room and invitee: %w", err)
	}

	return toInvitationDomain(rec), nil
}

func (r *ChatInvitationRepository) FindPendingByRoomAndInviter(
	ctx context.Context,
	roomID chatroom.ID,
	inviterID participant.ID,
) (*chatinvitation.ChatInvitation, error) {
	query, args, err := ChatInvitationTable.Select(ChatInvitationTable.Columns...).
		Where(squirrel.And{
			squirrel.Eq{"room_id": roomID, "inviter_id": inviterID, "status": chatinvitation.Pending},
			notExpired,
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build chat invitation query: %w", err)
	}

	rec, err := postgres.ScanOne[ChatInvitationRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, chatinvitation.ErrNotFound
		}
		return nil, fmt.Errorf("find chat invitation by room and inviter: %w", err)
	}

	return toInvitationDomain(rec), nil
}

func (r *ChatInvitationRepository) FindByRoom(ctx context.Context, roomID chatroom.ID) ([]*chatinvitation.ChatInvitation, error) {
	query, args, err := ChatInvitationTable.Select(ChatInvitationTable.Columns...).
		Where(squirrel.Eq{"room_id": roomID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build chat invitations query: %w", err)
	}

	records, err := postgres.ScanAll[ChatInvitationRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("scan chat invitations: %w", err)
	}

	invitations := make([]*chatinvitation.ChatInvitation, 0, len(records))
	for _, rec := range records {
		invitations = append(invitations, toInvitationDomain(&rec))
	}
	return invitations, nil
}

func (r *ChatInvitationRepository) UpdateStatus(ctx context.Context, id chatinvitation.ID, status chatinvitation.Status) error {
	query, args, err := ChatInvitationTable.Update().
		Set("status", status).
		Set("updated_at", squirrel.Expr("now()")).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update chat invitation: %w", err)
	}

	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

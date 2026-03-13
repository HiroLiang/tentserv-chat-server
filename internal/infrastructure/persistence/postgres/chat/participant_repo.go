package chat

import (
	"context"
	"errors"
	"fmt"

	"github.com/HiroLiang/goat-server/internal/domain/agent"
	"github.com/HiroLiang/goat-server/internal/domain/participant"
	"github.com/HiroLiang/goat-server/internal/domain/user"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var ParticipantTable = postgres.Table{
	Name: "public.participants",
	Columns: []string{
		"id",
		"type",
		"user_id",
		"agent_id",
		"display_name",
		"avatar_name",
		"created_at",
	},
}

type ParticipantRepository struct {
	db *sqlx.DB
}

var _ participant.Repository = (*ParticipantRepository)(nil)

func NewParticipantRepository(db *sqlx.DB) *ParticipantRepository {
	return &ParticipantRepository{db: db}
}

func (r *ParticipantRepository) FindByID(ctx context.Context, id participant.ID) (*participant.Participant, error) {
	return r.findOneBy(ctx, squirrel.Eq{"id": id})
}

func (r *ParticipantRepository) FindByUserID(ctx context.Context, userID user.ID) (*participant.Participant, error) {
	return r.findOneBy(ctx, squirrel.Eq{"user_id": userID, "type": participant.UserType})
}

func (r *ParticipantRepository) FindByAgentID(ctx context.Context, agentID agent.ID) (*participant.Participant, error) {
	return r.findOneBy(ctx, squirrel.Eq{"agent_id": agentID, "type": participant.AgentType})
}

func (r *ParticipantRepository) FindSystem(ctx context.Context) (*participant.Participant, error) {
	return r.findOneBy(ctx, squirrel.Eq{"type": participant.SystemType})
}

func (r *ParticipantRepository) Create(ctx context.Context, p *participant.Participant) error {
	rec := toParticipantRecordRecord(p)

	query, args, err := ParticipantTable.Insert().
		Columns("type", "user_id", "agent_id", "display_name", "avatar_name").
		Values(rec.Type, rec.UserID, rec.AgentID, rec.DisplayName, rec.AvatarName).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert participant: %w", err)
	}

	return postgres.Exec(ctx, r.db, query, args...)
}

func (r *ParticipantRepository) findOneBy(
	ctx context.Context,
	cond squirrel.Sqlizer,
) (*participant.Participant, error) {

	query, args, err := ParticipantTable.Select(ParticipantTable.Columns...).
		Where(cond).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build participant query: %w", err)
	}

	rec, err := postgres.ScanOne[ParticipantRecord](ctx, r.db, query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, participant.ErrNotFound
		}
		return nil, fmt.Errorf("find participant: %w", err)
	}

	return toParticipantDomain(rec)
}

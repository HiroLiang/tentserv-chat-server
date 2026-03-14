package chat

import (
	"context"
	"errors"
	"fmt"

	"github.com/HiroLiang/goat-server/internal/domain/participant"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres"
	"github.com/jmoiron/sqlx"
)

type ParticipantRepository struct {
	db *sqlx.DB
}

var _ participant.Repository = (*ParticipantRepository)(nil)

func NewParticipantRepository(db *sqlx.DB) *ParticipantRepository {
	return &ParticipantRepository{db: db}
}

func (r *ParticipantRepository) FindByID(ctx context.Context, id participant.ID) (*participant.Participant, error) {
	query := `
		SELECT p.id, p.type, pu.user_id, pa.agent_id, ps.system_type, p.created_at
		FROM public.participants p
		LEFT JOIN public.participant_users pu ON pu.participant_id = p.id
		LEFT JOIN public.participant_agents pa ON pa.participant_id = p.id
		LEFT JOIN public.participant_systems ps ON ps.participant_id = p.id
		WHERE p.id = $1
		LIMIT 1`

	rec, err := postgres.ScanOne[ParticipantRecord](ctx, r.db, query, id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, participant.ErrNotFound
		}
		return nil, fmt.Errorf("find participant: %w", err)
	}

	return toParticipantDomain(rec)
}

func (r *ParticipantRepository) FindByUserID(ctx context.Context, userID shared.UserID) (*participant.Participant, error) {
	query := `
		SELECT p.id, p.type, pu.user_id, NULL::bigint AS agent_id, NULL::text AS system_type, p.created_at
		FROM public.participants p
		JOIN public.participant_users pu ON pu.participant_id = p.id
		WHERE pu.user_id = $1
		LIMIT 1`

	rec, err := postgres.ScanOne[ParticipantRecord](ctx, r.db, query, userID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, participant.ErrNotFound
		}
		return nil, fmt.Errorf("find participant by user: %w", err)
	}

	return toParticipantDomain(rec)
}

func (r *ParticipantRepository) FindByAgentID(ctx context.Context, agentID int64) (*participant.Participant, error) {
	query := `
		SELECT p.id, p.type, NULL::bigint AS user_id, pa.agent_id, NULL::text AS system_type, p.created_at
		FROM public.participants p
		JOIN public.participant_agents pa ON pa.participant_id = p.id
		WHERE pa.agent_id = $1
		LIMIT 1`

	rec, err := postgres.ScanOne[ParticipantRecord](ctx, r.db, query, agentID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, participant.ErrNotFound
		}
		return nil, fmt.Errorf("find participant by agent: %w", err)
	}

	return toParticipantDomain(rec)
}

func (r *ParticipantRepository) FindSystem(ctx context.Context) (*participant.Participant, error) {
	query := `
		SELECT p.id, p.type, NULL::bigint AS user_id, NULL::bigint AS agent_id, ps.system_type, p.created_at
		FROM public.participants p
		JOIN public.participant_systems ps ON ps.participant_id = p.id
		LIMIT 1`

	rec, err := postgres.ScanOne[ParticipantRecord](ctx, r.db, query)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, participant.ErrNotFound
		}
		return nil, fmt.Errorf("find system participant: %w", err)
	}

	return toParticipantDomain(rec)
}

func (r *ParticipantRepository) Create(ctx context.Context, p *participant.Participant) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Insert into participants
	var id participant.ID
	err = tx.QueryRowContext(ctx,
		`INSERT INTO public.participants (type) VALUES ($1) RETURNING id`,
		p.Type,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("insert participant: %w", err)
	}
	p.ID = id

	// Insert into the corresponding join table
	switch p.Type {
	case participant.UserType:
		if p.UserID == nil {
			return fmt.Errorf("user_id required for user participant")
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO public.participant_users (participant_id, user_id) VALUES ($1, $2)`,
			id, *p.UserID,
		)
	case participant.AgentType:
		if p.AgentID == nil {
			return fmt.Errorf("agent_id required for agent participant")
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO public.participant_agents (participant_id, agent_id) VALUES ($1, $2)`,
			id, *p.AgentID,
		)
	case participant.SystemType:
		if p.SystemType == nil {
			return fmt.Errorf("system_type required for system participant")
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO public.participant_systems (participant_id, system_type) VALUES ($1, $2)`,
			id, *p.SystemType,
		)
	default:
		return fmt.Errorf("unknown participant type: %s", p.Type)
	}
	if err != nil {
		return fmt.Errorf("insert participant detail: %w", err)
	}

	return tx.Commit()
}

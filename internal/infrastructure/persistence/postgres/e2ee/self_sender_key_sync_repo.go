package e2ee

import (
	"context"
	"errors"
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var selfSenderKeySyncTable = postgres.Table{
	Name: "public.participant_self_sender_key_syncs",
	Columns: []string{
		"id",
		"participant_id",
		"requester_device_id",
		"provider_device_id",
		"status",
		"requested_at",
		"provider_claimed_at",
		"uploaded_at",
		"completed_at",
		"failed_at",
		"last_error",
		"updated_at",
	},
}

type SelfSenderKeySyncRepository struct {
	postgres.BaseRepo
}

var _ selfsenderkeysync.Repository = (*SelfSenderKeySyncRepository)(nil)

func NewSelfSenderKeySyncRepository(db *sqlx.DB) *SelfSenderKeySyncRepository {
	return &SelfSenderKeySyncRepository{BaseRepo: postgres.NewBaseRepo(db)}
}

func (r *SelfSenderKeySyncRepository) FindByParticipantID(
	ctx context.Context,
	participantID participant.ID,
) (*selfsenderkeysync.SelfSenderKeySync, error) {
	query, args, err := selfSenderKeySyncTable.Select(selfSenderKeySyncTable.Columns...).
		Where(squirrel.Eq{"participant_id": int64(participantID)}).
		OrderBy("id DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build self sender key sync query: %w", err)
	}

	record, err := postgres.ScanOne[SelfSenderKeySyncRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, selfsenderkeysync.ErrNotFound
		}
		return nil, fmt.Errorf("find self sender key sync: %w", err)
	}
	return toSelfSenderKeySyncDomain(record)
}

func (r *SelfSenderKeySyncRepository) UpsertPending(
	ctx context.Context,
	participantID participant.ID,
	requesterDeviceID shared.DeviceID,
) (*selfsenderkeysync.SelfSenderKeySync, error) {
	const query = `
INSERT INTO public.participant_self_sender_key_syncs
    (participant_id, requester_device_id, provider_device_id, status, requested_at, provider_claimed_at, uploaded_at, completed_at, failed_at, last_error, updated_at)
VALUES ($1, $2, NULL, 'pending_provider', now(), NULL, NULL, NULL, NULL, NULL, now())
RETURNING id, participant_id, requester_device_id, provider_device_id, status, requested_at, provider_claimed_at, uploaded_at, completed_at, failed_at, last_error, updated_at`

	row := r.GetDB(ctx).QueryRowxContext(ctx, query, int64(participantID), requesterDeviceID.String())
	var record SelfSenderKeySyncRecord
	if err := row.StructScan(&record); err != nil {
		return nil, fmt.Errorf("upsert self sender key sync: %w", err)
	}
	return toSelfSenderKeySyncDomain(&record)
}

func (r *SelfSenderKeySyncRepository) ClaimProvider(
	ctx context.Context,
	id selfsenderkeysync.ID,
	participantID participant.ID,
	requesterDeviceID, providerDeviceID shared.DeviceID,
) (bool, error) {
const query = `
UPDATE public.participant_self_sender_key_syncs
SET provider_device_id = $4,
    status = 'syncing',
    provider_claimed_at = now(),
    failed_at = NULL,
    last_error = NULL,
    updated_at = now()
WHERE id = $1
  AND participant_id = $2
  AND requester_device_id = $3
  AND status = 'pending_provider'
  AND provider_device_id IS NULL`

	result, err := r.GetDB(ctx).ExecContext(ctx, query, int64(id), int64(participantID), requesterDeviceID.String(), providerDeviceID.String())
	if err != nil {
		return false, fmt.Errorf("claim self sender key provider: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("claim self sender key provider rows affected: %w", err)
	}
	return rows > 0, nil
}

func (r *SelfSenderKeySyncRepository) MarkUploaded(
	ctx context.Context,
	id selfsenderkeysync.ID,
	participantID participant.ID,
	requesterDeviceID, providerDeviceID shared.DeviceID,
) error {
	return postgres.Exec(
		ctx,
		r.GetDB(ctx),
		`UPDATE public.participant_self_sender_key_syncs
SET status = 'uploaded',
    uploaded_at = now(),
    failed_at = NULL,
    last_error = NULL,
    updated_at = now()
WHERE id = $1
  AND participant_id = $2
  AND requester_device_id = $3
  AND provider_device_id = $4
  AND status = 'syncing'`,
		int64(id),
		int64(participantID),
		requesterDeviceID.String(),
		providerDeviceID.String(),
	)
}

func (r *SelfSenderKeySyncRepository) MarkCompleted(
	ctx context.Context,
	id selfsenderkeysync.ID,
	participantID participant.ID,
	requesterDeviceID shared.DeviceID,
) error {
	return postgres.Exec(
		ctx,
		r.GetDB(ctx),
		`UPDATE public.participant_self_sender_key_syncs
SET status = 'completed',
    completed_at = now(),
    failed_at = NULL,
    last_error = NULL,
    updated_at = now()
WHERE id = $1
  AND participant_id = $2
  AND requester_device_id = $3
  AND status IN ('uploaded', 'completed')`,
		int64(id),
		int64(participantID),
		requesterDeviceID.String(),
	)
}

func (r *SelfSenderKeySyncRepository) MarkFailed(
	ctx context.Context,
	id selfsenderkeysync.ID,
	participantID participant.ID,
	requesterDeviceID, providerDeviceID shared.DeviceID,
	lastError string,
	retryable bool,
) error {
	status := "failed"
	providerValue := providerDeviceID.String()
	if retryable {
		status = "pending_provider"
		providerValue = ""
	}

	query := `
UPDATE public.participant_self_sender_key_syncs
SET provider_device_id = CASE WHEN $4 = 'pending_provider' THEN NULL ELSE provider_device_id END,
    status = $4,
    failed_at = now(),
    last_error = NULLIF($5, ''),
    updated_at = now()
WHERE id = $1
  AND participant_id = $2
  AND requester_device_id = $3`
	args := []any{int64(id), int64(participantID), requesterDeviceID.String(), status, lastError}
	if providerValue != "" {
		query += `
  AND provider_device_id = $6`
		args = append(args, providerValue)
	}

	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

func toSelfSenderKeySyncDomain(record *SelfSenderKeySyncRecord) (*selfsenderkeysync.SelfSenderKeySync, error) {
	if record == nil {
		return nil, selfsenderkeysync.ErrNotFound
	}

	requesterDeviceID, err := shared.ParseDeviceID(record.RequesterDeviceID)
	if err != nil {
		return nil, fmt.Errorf("parse requester device id: %w", err)
	}

	var providerDeviceID *shared.DeviceID
	if record.ProviderDeviceID != nil && *record.ProviderDeviceID != "" {
		parsed, err := shared.ParseDeviceID(*record.ProviderDeviceID)
		if err != nil {
			return nil, fmt.Errorf("parse provider device id: %w", err)
		}
		providerDeviceID = &parsed
	}

	return &selfsenderkeysync.SelfSenderKeySync{
		ID:                selfsenderkeysync.ID(record.ID),
		ParticipantID:     participant.ID(record.ParticipantID),
		RequesterDeviceID: requesterDeviceID,
		ProviderDeviceID:  providerDeviceID,
		Status:            selfsenderkeysync.Status(record.Status),
		RequestedAt:       record.RequestedAt,
		ProviderClaimedAt: record.ProviderClaimedAt,
		UploadedAt:        record.UploadedAt,
		CompletedAt:       record.CompletedAt,
		FailedAt:          record.FailedAt,
		LastError:         record.LastError,
		UpdatedAt:         record.UpdatedAt,
	}, nil
}

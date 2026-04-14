package e2ee

import (
	"context"
	"errors"
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysyncdistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var selfSenderKeySyncDistributionTable = postgres.Table{
	Name: "public.self_sender_key_sync_distributions",
	Columns: []string{
		"id",
		"self_sender_key_sync_id",
		"participant_id",
		"requester_device_id",
		"provider_device_id",
		"sender_member_id",
		"sender_device_id",
		"sender_key_version",
		"distribution_message",
		"status",
		"created_at",
		"consumed_at",
		"failed_at",
	},
}

type SelfSenderKeySyncDistributionRepository struct {
	postgres.BaseRepo
}

var _ selfsenderkeysyncdistribution.Repository = (*SelfSenderKeySyncDistributionRepository)(nil)

func NewSelfSenderKeySyncDistributionRepository(db *sqlx.DB) *SelfSenderKeySyncDistributionRepository {
	return &SelfSenderKeySyncDistributionRepository{BaseRepo: postgres.NewBaseRepo(db)}
}

func (r *SelfSenderKeySyncDistributionRepository) ReplaceForSync(
	ctx context.Context,
	selfSyncID selfsenderkeysync.ID,
	participantID participant.ID,
	requesterDeviceID shared.DeviceID,
	providerDeviceID shared.DeviceID,
	rows []*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution,
) error {
	if err := postgres.Exec(
		ctx,
		r.GetDB(ctx),
		`DELETE FROM public.self_sender_key_sync_distributions
WHERE self_sender_key_sync_id = $1
  AND participant_id = $2
  AND requester_device_id = $3
  AND provider_device_id = $4`,
		int64(selfSyncID),
		int64(participantID),
		requesterDeviceID.String(),
		providerDeviceID.String(),
	); err != nil {
		return fmt.Errorf("delete self sender key sync distributions: %w", err)
	}

	for _, row := range rows {
		if err := postgres.Exec(
			ctx,
			r.GetDB(ctx),
			`INSERT INTO public.self_sender_key_sync_distributions
    (self_sender_key_sync_id, participant_id, requester_device_id, provider_device_id, sender_member_id, sender_device_id, sender_key_version, distribution_message, status, created_at, consumed_at, failed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'available', now(), NULL, NULL)`,
			int64(selfSyncID),
			int64(participantID),
			requesterDeviceID.String(),
			providerDeviceID.String(),
			int64(row.SenderMemberID),
			row.SenderDeviceID.String(),
			row.SenderKeyVersion,
			row.DistributionMessage,
		); err != nil {
			return fmt.Errorf("insert self sender key sync distribution: %w", err)
		}
	}

	return nil
}

func (r *SelfSenderKeySyncDistributionRepository) FindPendingByRequester(
	ctx context.Context,
	selfSyncID selfsenderkeysync.ID,
	participantID participant.ID,
	requesterDeviceID shared.DeviceID,
) ([]*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution, error) {
	query, args, err := selfSenderKeySyncDistributionTable.Select(selfSenderKeySyncDistributionTable.Columns...).
		Where(squirrel.Eq{
			"self_sender_key_sync_id": int64(selfSyncID),
			"participant_id":          int64(participantID),
			"requester_device_id": requesterDeviceID.String(),
			"status":             string(selfsenderkeysyncdistribution.StatusAvailable),
		}).
		OrderBy("id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build self sender key sync distribution query: %w", err)
	}

	records, err := postgres.ScanAll[SelfSenderKeySyncDistributionRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("find pending self sender key sync distributions: %w", err)
	}

	items := make([]*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution, 0, len(records))
	for _, record := range records {
		item, convErr := toSelfSenderKeySyncDistributionDomain(&record)
		if convErr != nil {
			return nil, convErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *SelfSenderKeySyncDistributionRepository) MarkConsumed(
	ctx context.Context,
	selfSyncID selfsenderkeysync.ID,
	id selfsenderkeysyncdistribution.ID,
	participantID participant.ID,
	requesterDeviceID shared.DeviceID,
) error {
	return postgres.Exec(
		ctx,
		r.GetDB(ctx),
		`UPDATE public.self_sender_key_sync_distributions
SET status = 'consumed',
    consumed_at = now(),
    failed_at = NULL
WHERE id = $1
  AND self_sender_key_sync_id = $2
  AND participant_id = $3
  AND requester_device_id = $4`,
		int64(id),
		int64(selfSyncID),
		int64(participantID),
		requesterDeviceID.String(),
	)
}

func (r *SelfSenderKeySyncDistributionRepository) MarkFailed(
	ctx context.Context,
	selfSyncID selfsenderkeysync.ID,
	id selfsenderkeysyncdistribution.ID,
	participantID participant.ID,
	requesterDeviceID shared.DeviceID,
) error {
	return postgres.Exec(
		ctx,
		r.GetDB(ctx),
		`UPDATE public.self_sender_key_sync_distributions
SET status = 'failed',
    failed_at = now(),
    consumed_at = NULL
WHERE id = $1
  AND self_sender_key_sync_id = $2
  AND participant_id = $3
  AND requester_device_id = $4`,
		int64(id),
		int64(selfSyncID),
		int64(participantID),
		requesterDeviceID.String(),
	)
}

func (r *SelfSenderKeySyncDistributionRepository) HasNonConsumed(
	ctx context.Context,
	selfSyncID selfsenderkeysync.ID,
	participantID participant.ID,
	requesterDeviceID shared.DeviceID,
) (bool, error) {
	query, args, err := squirrel.
		Select("COUNT(1) AS count").
		From(selfSenderKeySyncDistributionTable.Name).
		Where(squirrel.Eq{
			"self_sender_key_sync_id": int64(selfSyncID),
			"participant_id":          int64(participantID),
			"requester_device_id": requesterDeviceID.String(),
		}).
		Where(squirrel.NotEq{"status": string(selfsenderkeysyncdistribution.StatusConsumed)}).
		Limit(1).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build self sender key sync non-consumed query: %w", err)
	}

	record, err := postgres.ScanOne[struct {
		Count int `db:"count"`
	}](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("check self sender key sync non-consumed rows: %w", err)
	}
	return record != nil && record.Count > 0, nil
}

func toSelfSenderKeySyncDistributionDomain(
	record *SelfSenderKeySyncDistributionRecord,
) (*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution, error) {
	if record == nil {
		return nil, selfsenderkeysyncdistribution.ErrNotFound
	}

	requesterDeviceID, err := shared.ParseDeviceID(record.RequesterDeviceID)
	if err != nil {
		return nil, fmt.Errorf("parse requester device id: %w", err)
	}
	providerDeviceID, err := shared.ParseDeviceID(record.ProviderDeviceID)
	if err != nil {
		return nil, fmt.Errorf("parse provider device id: %w", err)
	}
	senderDeviceID, err := shared.ParseDeviceID(record.SenderDeviceID)
	if err != nil {
		return nil, fmt.Errorf("parse sender device id: %w", err)
	}

	return &selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution{
		ID:                  selfsenderkeysyncdistribution.ID(record.ID),
		SelfSenderKeySyncID: selfsenderkeysync.ID(record.SelfSenderKeySyncID),
		ParticipantID:       participant.ID(record.ParticipantID),
		RequesterDeviceID:   requesterDeviceID,
		ProviderDeviceID:    providerDeviceID,
		SenderMemberID:      chatmember.ID(record.SenderMemberID),
		SenderDeviceID:      senderDeviceID,
		SenderKeyVersion:    record.SenderKeyVersion,
		DistributionMessage: append([]byte(nil), record.DistributionMessage...),
		Status:              selfsenderkeysyncdistribution.Status(record.Status),
		CreatedAt:           record.CreatedAt,
		ConsumedAt:          record.ConsumedAt,
		FailedAt:            record.FailedAt,
	}, nil
}

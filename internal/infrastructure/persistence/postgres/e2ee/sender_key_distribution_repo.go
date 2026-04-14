package e2ee

import (
	"context"
	"errors"
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
)

var distributionTable = postgres.Table{
	Name: "public.sender_key_distributions",
	Columns: []string{
		"id",
		"sender_member_id",
		"sender_device_id",
		"receiver_member_id",
		"receiver_device_id",
		"sender_key_version",
		"chain_id",
		"distribution_message",
		"status",
		"distributed_at",
		"consumed_at",
		"failed_at",
	},
}

type SenderKeyDistributionRepository struct {
	postgres.BaseRepo
}

var _ senderkeydistribution.Repository = (*SenderKeyDistributionRepository)(nil)

func NewSenderKeyDistributionRepository(db *sqlx.DB) *SenderKeyDistributionRepository {
	return &SenderKeyDistributionRepository{BaseRepo: postgres.NewBaseRepo(db)}
}

func (r *SenderKeyDistributionRepository) UpsertBatch(
	ctx context.Context,
	dists []*senderkeydistribution.SenderKeyDistribution,
) error {
	if len(dists) == 0 {
		return nil
	}

	q := distributionTable.Insert().
		Columns("sender_member_id", "sender_device_id", "receiver_member_id", "receiver_device_id", "sender_key_version", "chain_id", "distribution_message", "status").
		Suffix(`ON CONFLICT (sender_member_id, sender_device_id, receiver_member_id, receiver_device_id)
			DO UPDATE SET sender_key_version = GREATEST(EXCLUDED.sender_key_version, sender_key_distributions.sender_key_version),
			              chain_id = GREATEST(EXCLUDED.chain_id, sender_key_distributions.chain_id),
			              status = CASE
			                  WHEN EXCLUDED.sender_key_version >= sender_key_distributions.sender_key_version THEN EXCLUDED.status
			                  ELSE sender_key_distributions.status
			              END,
			              consumed_at = CASE
			                  WHEN EXCLUDED.status = 'consumed' THEN now()
			                  ELSE sender_key_distributions.consumed_at
			              END,
			              failed_at = CASE
			                  WHEN EXCLUDED.status = 'failed' THEN now()
			                  ELSE sender_key_distributions.failed_at
			              END,
			              distributed_at = now()`)

	for _, d := range dists {
		version := d.SenderKeyVersion
		if version == 0 {
			version = int64(d.ChainID)
		}
		chainID := d.ChainID
		if chainID == 0 {
			chainID = version
		}
		message := d.DistributionMessage
		if message == nil {
			message = []byte{}
		}
		q = q.Values(
			int64(d.SenderMemberID),
			d.SenderDeviceID.String(),
			int64(d.ReceiverMemberID),
			d.ReceiverDeviceID.String(),
			version,
			chainID,
			message,
			senderkeydistribution.StatusConsumed,
		)
	}

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("build upsert distributions: %w", err)
	}

	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

func (r *SenderKeyDistributionRepository) FindPendingReceivers(
	ctx context.Context,
	senderMemberID chatmember.ID,
	senderDeviceID shared.DeviceID,
	latestChainID int64,
) ([]chatmember.ID, error) {
	const query = `
SELECT cm.id
FROM public.chat_members cm
WHERE cm.room_id = (SELECT room_id FROM public.chat_members WHERE id = $1)
  AND cm.id != $1
  AND cm.is_deleted = false
  AND NOT EXISTS (
      SELECT 1 FROM public.sender_key_distributions skd
      WHERE skd.sender_member_id = $1
        AND skd.sender_device_id = $2
        AND skd.receiver_member_id = cm.id
        AND skd.sender_key_version >= $3
        AND skd.status = 'consumed'
  )`

	db := r.GetDB(ctx)
	var ids []int64
	if err := sqlx.SelectContext(ctx, db, &ids, query, int64(senderMemberID), senderDeviceID.String(), latestChainID); err != nil {
		return nil, fmt.Errorf("find pending receivers: %w", err)
	}

	result := make([]chatmember.ID, len(ids))
	for i, id := range ids {
		result[i] = chatmember.ID(id)
	}
	return result, nil
}

func (r *SenderKeyDistributionRepository) UpsertAvailable(
	ctx context.Context,
	dist *senderkeydistribution.SenderKeyDistribution,
) error {
	if dist.ChainID == 0 {
		dist.ChainID = dist.SenderKeyVersion
	}

	query := `
INSERT INTO public.sender_key_distributions
    (sender_member_id, sender_device_id, receiver_member_id, receiver_device_id, sender_key_version, chain_id, distribution_message, status, distributed_at, consumed_at, failed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), NULL, NULL)
ON CONFLICT (sender_member_id, sender_device_id, receiver_member_id, receiver_device_id)
DO UPDATE SET sender_key_version = EXCLUDED.sender_key_version,
              chain_id = EXCLUDED.chain_id,
              distribution_message = EXCLUDED.distribution_message,
              status = EXCLUDED.status,
              distributed_at = now(),
              consumed_at = NULL,
              failed_at = NULL
WHERE sender_key_distributions.sender_key_version <= EXCLUDED.sender_key_version
RETURNING id, distributed_at`

	row := r.GetDB(ctx).QueryRowxContext(
		ctx,
		query,
		int64(dist.SenderMemberID),
		dist.SenderDeviceID.String(),
		int64(dist.ReceiverMemberID),
		dist.ReceiverDeviceID.String(),
		dist.SenderKeyVersion,
		dist.ChainID,
		dist.DistributionMessage,
		senderkeydistribution.StatusAvailable,
	)
	if err := row.Scan(&dist.ID, &dist.DistributedAt); err != nil {
		return fmt.Errorf("upsert sender key distribution: %w", err)
	}
	dist.Status = senderkeydistribution.StatusAvailable
	return nil
}

func (r *SenderKeyDistributionRepository) FindLatest(
	ctx context.Context,
	senderMemberID chatmember.ID,
	senderDeviceID shared.DeviceID,
	receiverMemberID chatmember.ID,
	receiverDeviceID shared.DeviceID,
) (*senderkeydistribution.SenderKeyDistribution, error) {
	builder := distributionTable.Select(distributionTable.Columns...).
		Where("sender_member_id = ? AND sender_device_id = ? AND receiver_member_id = ? AND receiver_device_id = ?", int64(senderMemberID), senderDeviceID.String(), int64(receiverMemberID), receiverDeviceID.String())

	query, args, err := builder.
		OrderBy("sender_key_version DESC").
		Limit(1).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build latest distribution query: %w", err)
	}

	rec, err := postgres.ScanOne[SenderKeyDistributionRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, senderkeydistribution.ErrNotFound
		}
		return nil, fmt.Errorf("find latest distribution: %w", err)
	}
	return toDistributionDomain(rec), nil
}

func (r *SenderKeyDistributionRepository) FindLatestForReceiver(
	ctx context.Context,
	senderMemberID chatmember.ID,
	receiverMemberID chatmember.ID,
	receiverDeviceID shared.DeviceID,
) (*senderkeydistribution.SenderKeyDistribution, error) {
	builder := distributionTable.Select(distributionTable.Columns...).
		Where(
			"sender_member_id = ? AND receiver_member_id = ? AND receiver_device_id = ?",
			int64(senderMemberID),
			int64(receiverMemberID),
			receiverDeviceID.String(),
		)

	query, args, err := builder.
		OrderBy("sender_key_version DESC", "distributed_at DESC", "id DESC").
		Limit(1).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build latest distribution by receiver query: %w", err)
	}

	rec, err := postgres.ScanOne[SenderKeyDistributionRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, senderkeydistribution.ErrNotFound
		}
		return nil, fmt.Errorf("find latest distribution by receiver: %w", err)
	}
	return toDistributionDomain(rec), nil
}

func (r *SenderKeyDistributionRepository) FindAvailableByRoomAndReceiver(
	ctx context.Context,
	roomID chatroom.ID,
	receiverMemberID chatmember.ID,
	receiverDeviceID shared.DeviceID,
) ([]*senderkeydistribution.SenderKeyDistribution, error) {
	query := `
SELECT skd.id,
       skd.sender_member_id,
       skd.sender_device_id,
       skd.receiver_member_id,
       skd.receiver_device_id,
       skd.sender_key_version,
       skd.chain_id,
       skd.distribution_message,
       skd.status,
       skd.distributed_at,
       skd.consumed_at,
       skd.failed_at
FROM public.sender_key_distributions skd
JOIN public.chat_members cm_sender ON cm_sender.id = skd.sender_member_id
JOIN public.chat_members cm_receiver ON cm_receiver.id = skd.receiver_member_id
WHERE cm_sender.room_id = $1
  AND cm_receiver.room_id = $1
  AND cm_sender.is_deleted = false
  AND cm_receiver.is_deleted = false
  AND skd.receiver_member_id = $2
  AND skd.status = 'available'`

	args := []any{int64(roomID), int64(receiverMemberID)}
	if uuid.UUID(receiverDeviceID) != uuid.Nil {
		query += `
  AND skd.receiver_device_id = $3`
		args = append(args, receiverDeviceID.String())
	}
	query += `
ORDER BY skd.distributed_at ASC`

	var rows []SenderKeyDistributionRecord
	if err := sqlx.SelectContext(ctx, r.GetDB(ctx), &rows, query, args...); err != nil {
		return nil, fmt.Errorf("find available distributions: %w", err)
	}

	out := make([]*senderkeydistribution.SenderKeyDistribution, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDistributionDomain(&row))
	}
	return out, nil
}

func (r *SenderKeyDistributionRepository) FindByID(
	ctx context.Context,
	id senderkeydistribution.ID,
) (*senderkeydistribution.SenderKeyDistribution, error) {
	query, args, err := distributionTable.Select(distributionTable.Columns...).
		Where("id = ?", int64(id)).
		Limit(1).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build distribution by id query: %w", err)
	}

	rec, err := postgres.ScanOne[SenderKeyDistributionRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, senderkeydistribution.ErrNotFound
		}
		return nil, fmt.Errorf("find distribution by id: %w", err)
	}
	return toDistributionDomain(rec), nil
}

func (r *SenderKeyDistributionRepository) MarkConsumed(
	ctx context.Context,
	id senderkeydistribution.ID,
) error {
	const query = `
UPDATE public.sender_key_distributions
SET status = 'consumed',
    consumed_at = now(),
    failed_at = NULL
WHERE id = $1`
	if err := postgres.Exec(ctx, r.GetDB(ctx), query, int64(id)); err != nil {
		return fmt.Errorf("mark sender key distribution consumed: %w", err)
	}
	return nil
}

func (r *SenderKeyDistributionRepository) MarkFailed(
	ctx context.Context,
	id senderkeydistribution.ID,
) error {
	const query = `
UPDATE public.sender_key_distributions
SET status = 'failed',
    failed_at = now()
WHERE id = $1`
	if err := postgres.Exec(ctx, r.GetDB(ctx), query, int64(id)); err != nil {
		return fmt.Errorf("mark sender key distribution failed: %w", err)
	}
	return nil
}

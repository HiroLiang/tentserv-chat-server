package deliveryqueue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/deliveryqueue"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var deliveryQueueTable = postgres.Table{
	Name: "public.delivery_queue",
	Columns: []string{
		"id", "user_id", "payload_type", "payload", "status", "created_at", "delivered_at",
	},
}

type DeliveryQueueRepository struct {
	postgres.BaseRepo
}

var _ deliveryqueue.Repository = (*DeliveryQueueRepository)(nil)

func NewDeliveryQueueRepository(db *sqlx.DB) *DeliveryQueueRepository {
	return &DeliveryQueueRepository{BaseRepo: postgres.NewBaseRepo(db)}
}

func (r *DeliveryQueueRepository) Enqueue(ctx context.Context, item *deliveryqueue.DeliveryQueue) error {
	query, args, err := deliveryQueueTable.Insert().
		Columns("user_id", "payload_type", "payload", "status").
		Values(item.UserID, item.PayloadType, item.Payload, deliveryqueue.StatusPending).
		Suffix("RETURNING id, created_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("build enqueue query: %w", err)
	}

	db := r.GetDB(ctx)
	row := db.QueryRowxContext(ctx, query, args...)
	if err := row.Scan(&item.ID, &item.CreatedAt); err != nil {
		return fmt.Errorf("enqueue delivery item: %w", err)
	}
	item.Status = deliveryqueue.StatusPending
	return nil
}

func (r *DeliveryQueueRepository) MarkDelivered(ctx context.Context, id deliveryqueue.ID) error {
	query, args, err := deliveryQueueTable.Update().
		Set("status", deliveryqueue.StatusDelivered).
		Set("delivered_at", squirrel.Expr("now()")).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build mark delivered query: %w", err)
	}

	db := r.GetDB(ctx)
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("mark delivered: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark delivered rows affected: %w", err)
	}
	if rows == 0 {
		return deliveryqueue.ErrNotFound
	}
	return nil
}

func (r *DeliveryQueueRepository) FindPendingOlderThan(ctx context.Context, age time.Duration) ([]*deliveryqueue.DeliveryQueue, error) {
	cutoff := time.Now().Add(-age)
	query, args, err := deliveryQueueTable.Select(deliveryQueueTable.Columns...).
		Where(squirrel.And{
			squirrel.Eq{"status": deliveryqueue.StatusPending},
			squirrel.Lt{"created_at": cutoff},
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find pending query: %w", err)
	}

	records, err := postgres.ScanAll[DeliveryQueueRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan pending delivery items: %w", err)
	}

	items := make([]*deliveryqueue.DeliveryQueue, 0, len(records))
	for i := range records {
		items = append(items, toDomain(&records[i]))
	}
	return items, nil
}

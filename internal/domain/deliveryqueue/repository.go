package deliveryqueue

import (
	"context"
	"time"
)

type Repository interface {
	Enqueue(ctx context.Context, item *DeliveryQueue) error
	MarkDelivered(ctx context.Context, id ID) error
	FindPendingOlderThan(ctx context.Context, age time.Duration) ([]*DeliveryQueue, error)
}

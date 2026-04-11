package deliveryqueue

import (
	"context"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type Repository interface {
	Enqueue(ctx context.Context, item *DeliveryQueue) error
	MarkDelivered(ctx context.Context, id ID) error
	FindPendingByUser(ctx context.Context, userID shared.UserID) ([]*DeliveryQueue, error)
	FindPendingOlderThan(ctx context.Context, age time.Duration) ([]*DeliveryQueue, error)
}

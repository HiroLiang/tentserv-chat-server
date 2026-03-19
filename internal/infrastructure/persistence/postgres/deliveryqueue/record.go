package deliveryqueue

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/deliveryqueue"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type DeliveryQueueRecord struct {
	ID          deliveryqueue.ID          `db:"id"`
	UserID      shared.UserID             `db:"user_id"`
	PayloadType deliveryqueue.PayloadType `db:"payload_type"`
	Payload     []byte                    `db:"payload"`
	Status      deliveryqueue.Status      `db:"status"`
	CreatedAt   time.Time                 `db:"created_at"`
	DeliveredAt *time.Time                `db:"delivered_at"`
}

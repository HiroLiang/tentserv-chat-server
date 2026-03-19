package deliveryqueue

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type DeliveryQueue struct {
	ID          ID
	UserID      shared.UserID
	PayloadType PayloadType
	Payload     []byte
	Status      Status
	CreatedAt   time.Time
	DeliveredAt *time.Time
}

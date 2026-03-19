package push

import (
	"context"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

// Dispatcher is the abstraction for enqueue + push. Swap DB→Redis→Kafka here.
type Dispatcher interface {
	Dispatch(ctx context.Context, userID shared.UserID, msgType string, payload []byte) error
}

// DirectPusher is implemented by the WS Hub. Used by dispatcher and scheduler.
type DirectPusher interface {
	PushAndWaitAck(userID string, deliveryID int64, msgType string, payload []byte, timeout time.Duration) bool
	ResolveAck(deliveryID int64)
}

package push

import (
	"context"
	"fmt"
	"strconv"
	"time"

	appPush "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/push"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/deliveryqueue"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"go.uber.org/zap"
)

const pushTimeout = 5 * time.Second

// DBDispatcher persists the event to the delivery_queue table, then immediately
// attempts a push over WebSocket. If the client is offline, the scheduler will
// retry.
type DBDispatcher struct {
	repo   deliveryqueue.Repository
	pusher appPush.DirectPusher
}

var _ appPush.Dispatcher = (*DBDispatcher)(nil)

func NewDBDispatcher(repo deliveryqueue.Repository, pusher appPush.DirectPusher) *DBDispatcher {
	return &DBDispatcher{repo: repo, pusher: pusher}
}

func (d *DBDispatcher) Dispatch(ctx context.Context, userID shared.UserID, msgType string, payload []byte) error {
	item := &deliveryqueue.DeliveryQueue{
		UserID:      userID,
		PayloadType: deliveryqueue.PayloadType(msgType),
		Payload:     payload,
	}
	if err := d.repo.Enqueue(ctx, item); err != nil {
		return fmt.Errorf("dispatch enqueue: %w", err)
	}
	go d.pushAndTrack(context.Background(), userID, item)
	return nil
}

func (d *DBDispatcher) pushAndTrack(ctx context.Context, userID shared.UserID, item *deliveryqueue.DeliveryQueue) {
	userIDStr := strconv.FormatInt(int64(userID), 10)
	acked := d.pusher.PushAndWaitAck(userIDStr, int64(item.ID), string(item.PayloadType), item.Payload, pushTimeout)
	if acked {
		if err := d.repo.MarkDelivered(ctx, item.ID); err != nil {
			logger.Log.Error("mark delivered failed", zap.Int64("delivery_id", int64(item.ID)), zap.Error(err))
		}
	}
}

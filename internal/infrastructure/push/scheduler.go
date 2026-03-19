package push

import (
	"context"
	"strconv"
	"time"

	appPush "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/push"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/deliveryqueue"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"go.uber.org/zap"
)

const (
	schedulerInterval = 1 * time.Minute
	pendingAge        = 30 * time.Second
)

// RetryScheduler periodically re-pushes delivery items that remain pending.
type RetryScheduler struct {
	repo   deliveryqueue.Repository
	pusher appPush.DirectPusher
}

func NewRetryScheduler(repo deliveryqueue.Repository, pusher appPush.DirectPusher) *RetryScheduler {
	return &RetryScheduler{repo: repo, pusher: pusher}
}

func (s *RetryScheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(schedulerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.retry(ctx)
		}
	}
}

func (s *RetryScheduler) retry(ctx context.Context) {
	items, err := s.repo.FindPendingOlderThan(ctx, pendingAge)
	if err != nil {
		logger.Log.Error("scheduler find pending failed", zap.Error(err))
		return
	}

	for _, item := range items {
		go s.pushAndTrack(context.Background(), item)
	}
}

func (s *RetryScheduler) pushAndTrack(ctx context.Context, item *deliveryqueue.DeliveryQueue) {
	userIDStr := strconv.FormatInt(int64(item.UserID), 10)
	acked := s.pusher.PushAndWaitAck(userIDStr, int64(item.ID), string(item.PayloadType), item.Payload, pushTimeout)
	if acked {
		if err := s.repo.MarkDelivered(ctx, item.ID); err != nil {
			logger.Log.Error("scheduler mark delivered failed", zap.Int64("delivery_id", int64(item.ID)), zap.Error(err))
		}
	}
}

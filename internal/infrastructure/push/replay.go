package push

import (
	"context"
	"strconv"

	appPush "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/push"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/deliveryqueue"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"go.uber.org/zap"
)

func ReplayPendingForUser(
	ctx context.Context,
	repo deliveryqueue.Repository,
	pusher appPush.DirectPusher,
	userID shared.UserID,
) {
	items, err := repo.FindPendingByUser(ctx, userID)
	if err != nil {
		logger.Log.Error("replay pending deliveries failed", zap.Int64("user_id", int64(userID)), zap.Error(err))
		return
	}

	userIDStr := strconv.FormatInt(int64(userID), 10)
	for _, item := range items {
		acked := pusher.PushAndWaitAck(userIDStr, int64(item.ID), string(item.PayloadType), item.Payload, pushTimeout)
		if !acked {
			continue
		}
		if err := repo.MarkDelivered(ctx, item.ID); err != nil {
			logger.Log.Error("replay mark delivered failed", zap.Int64("delivery_id", int64(item.ID)), zap.Error(err))
		}
	}
}

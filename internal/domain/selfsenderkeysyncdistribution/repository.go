package selfsenderkeysyncdistribution

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type Repository interface {
	ReplaceForSync(
		ctx context.Context,
		selfSyncID selfsenderkeysync.ID,
		participantID participant.ID,
		requesterDeviceID shared.DeviceID,
		providerDeviceID shared.DeviceID,
		rows []*SelfSenderKeySyncDistribution,
	) error
	FindPendingByRequester(
		ctx context.Context,
		selfSyncID selfsenderkeysync.ID,
		participantID participant.ID,
		requesterDeviceID shared.DeviceID,
	) ([]*SelfSenderKeySyncDistribution, error)
	MarkConsumed(
		ctx context.Context,
		selfSyncID selfsenderkeysync.ID,
		id ID,
		participantID participant.ID,
		requesterDeviceID shared.DeviceID,
	) error
	MarkFailed(
		ctx context.Context,
		selfSyncID selfsenderkeysync.ID,
		id ID,
		participantID participant.ID,
		requesterDeviceID shared.DeviceID,
	) error
	HasNonConsumed(
		ctx context.Context,
		selfSyncID selfsenderkeysync.ID,
		participantID participant.ID,
		requesterDeviceID shared.DeviceID,
	) (bool, error)
}

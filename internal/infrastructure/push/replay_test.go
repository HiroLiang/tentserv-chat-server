package push

import (
	"context"
	"testing"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/deliveryqueue"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplayPendingForUser_ReplaysAndMarksAckedItemsDelivered(t *testing.T) {
	repo := &replayRepoStub{
		items: []*deliveryqueue.DeliveryQueue{
			{
				ID:          11,
				UserID:      42,
				PayloadType: deliveryqueue.PayloadTypeReplenishOTP,
				Payload:     []byte(`{"threshold":10}`),
				Status:      deliveryqueue.StatusPending,
			},
		},
	}
	pusher := &replayPusherStub{acked: map[int64]bool{11: true}}

	ReplayPendingForUser(context.Background(), repo, pusher, 42)

	require.Len(t, pusher.calls, 1)
	assert.Equal(t, "42", pusher.calls[0].userID)
	assert.Equal(t, int64(11), pusher.calls[0].deliveryID)
	assert.Equal(t, string(deliveryqueue.PayloadTypeReplenishOTP), pusher.calls[0].msgType)
	assert.Equal(t, []byte(`{"threshold":10}`), pusher.calls[0].payload)
	assert.Equal(t, []deliveryqueue.ID{11}, repo.markDeliveredCalls)
}

func TestReplayPendingForUser_KeepsUnackedItemsPending(t *testing.T) {
	repo := &replayRepoStub{
		items: []*deliveryqueue.DeliveryQueue{
			{
				ID:          12,
				UserID:      42,
				PayloadType: deliveryqueue.PayloadTypeMessage,
				Payload:     []byte(`{"body":"hello"}`),
				Status:      deliveryqueue.StatusPending,
			},
		},
	}
	pusher := &replayPusherStub{}

	ReplayPendingForUser(context.Background(), repo, pusher, 42)

	require.Len(t, pusher.calls, 1)
	assert.Empty(t, repo.markDeliveredCalls)
}

type replayRepoStub struct {
	items              []*deliveryqueue.DeliveryQueue
	markDeliveredCalls []deliveryqueue.ID
}

func (s *replayRepoStub) Create(context.Context, *deliveryqueue.DeliveryQueue) error {
	return nil
}

func (s *replayRepoStub) Enqueue(context.Context, *deliveryqueue.DeliveryQueue) error {
	return nil
}

func (s *replayRepoStub) MarkDelivered(_ context.Context, id deliveryqueue.ID) error {
	s.markDeliveredCalls = append(s.markDeliveredCalls, id)
	return nil
}

func (s *replayRepoStub) FindPendingByUser(_ context.Context, userID shared.UserID) ([]*deliveryqueue.DeliveryQueue, error) {
	out := make([]*deliveryqueue.DeliveryQueue, 0, len(s.items))
	for _, item := range s.items {
		if item.UserID != userID {
			continue
		}
		copied := *item
		out = append(out, &copied)
	}
	return out, nil
}

func (s *replayRepoStub) FindPendingOlderThan(context.Context, time.Duration) ([]*deliveryqueue.DeliveryQueue, error) {
	return nil, nil
}

type replayPusherStub struct {
	acked map[int64]bool
	calls []replayPushCall
}

type replayPushCall struct {
	userID     string
	deliveryID int64
	msgType    string
	payload    []byte
}

func (s *replayPusherStub) PushAndWaitAck(userID string, deliveryID int64, msgType string, payload []byte, _ time.Duration) bool {
	s.calls = append(s.calls, replayPushCall{
		userID:     userID,
		deliveryID: deliveryID,
		msgType:    msgType,
		payload:    append([]byte(nil), payload...),
	})
	return s.acked[deliveryID]
}

func (s *replayPusherStub) ResolveAck(int64) {}

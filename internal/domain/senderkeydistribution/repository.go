package senderkeydistribution

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
)

type Repository interface {
	// UpsertBatch records that receiverMemberID has fetched each sender's key at the given chain_id.
	// Uses ON CONFLICT (sender_member_id, receiver_member_id) DO UPDATE to update chain_id if newer.
	UpsertBatch(ctx context.Context, dists []*SenderKeyDistribution) error

	// FindPendingReceivers returns member IDs in the room who have not yet fetched
	// senderMemberID's key at latestChainID (or have never fetched it at all).
	FindPendingReceivers(ctx context.Context, senderMemberID chatmember.ID, latestChainID int) ([]chatmember.ID, error)

	UpsertAvailable(ctx context.Context, dist *SenderKeyDistribution) error
	FindLatest(ctx context.Context, senderMemberID, receiverMemberID chatmember.ID) (*SenderKeyDistribution, error)
	FindAvailableByRoomAndReceiver(ctx context.Context, roomID chatroom.ID, receiverMemberID chatmember.ID) ([]*SenderKeyDistribution, error)
	FindByID(ctx context.Context, id ID) (*SenderKeyDistribution, error)
	MarkConsumed(ctx context.Context, id ID) error
	MarkFailed(ctx context.Context, id ID) error
}

package senderkeydistribution

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
)

type Repository interface {
	// UpsertBatch records that receiverMemberID has fetched each sender's key at the given chain_id.
	// Uses ON CONFLICT (sender_member_id, receiver_member_id) DO UPDATE to update chain_id if newer.
	UpsertBatch(ctx context.Context, dists []*SenderKeyDistribution) error

	// FindPendingReceivers returns member IDs in the room who have not yet fetched
	// senderMemberID's key at latestChainID (or have never fetched it at all).
	FindPendingReceivers(ctx context.Context, senderMemberID chatmember.ID, latestChainID int) ([]chatmember.ID, error)
}

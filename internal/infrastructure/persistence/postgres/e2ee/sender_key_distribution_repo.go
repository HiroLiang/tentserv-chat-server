package e2ee

import (
	"context"
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/jmoiron/sqlx"
)

var distributionTable = postgres.Table{
	Name:    "public.sender_key_distributions",
	Columns: []string{"id", "sender_member_id", "receiver_member_id", "chain_id", "distributed_at"},
}

type SenderKeyDistributionRepository struct {
	postgres.BaseRepo
}

var _ senderkeydistribution.Repository = (*SenderKeyDistributionRepository)(nil)

func NewSenderKeyDistributionRepository(db *sqlx.DB) *SenderKeyDistributionRepository {
	return &SenderKeyDistributionRepository{BaseRepo: postgres.NewBaseRepo(db)}
}

// UpsertBatch records that each dist.ReceiverMemberID has fetched dist.SenderMemberID's key
// at dist.ChainID. Uses ON CONFLICT to update chain_id when a newer version is fetched.
func (r *SenderKeyDistributionRepository) UpsertBatch(
	ctx context.Context,
	dists []*senderkeydistribution.SenderKeyDistribution,
) error {
	if len(dists) == 0 {
		return nil
	}

	q := distributionTable.Insert().
		Columns("sender_member_id", "receiver_member_id", "chain_id").
		Suffix(`ON CONFLICT (sender_member_id, receiver_member_id)
			DO UPDATE SET chain_id = GREATEST(EXCLUDED.chain_id, sender_key_distributions.chain_id),
			              distributed_at = now()`)

	for _, d := range dists {
		q = q.Values(int64(d.SenderMemberID), int64(d.ReceiverMemberID), d.ChainID)
	}

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("build upsert distributions: %w", err)
	}

	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

// FindPendingReceivers returns the IDs of members in the same room as senderMemberID
// who have not yet fetched senderMemberID's key at latestChainID.
func (r *SenderKeyDistributionRepository) FindPendingReceivers(
	ctx context.Context,
	senderMemberID chatmember.ID,
	latestChainID int,
) ([]chatmember.ID, error) {
	const query = `
SELECT cm.id
FROM public.chat_members cm
WHERE cm.room_id = (SELECT room_id FROM public.chat_members WHERE id = $1)
  AND cm.id != $1
  AND cm.is_deleted = false
  AND NOT EXISTS (
      SELECT 1 FROM public.sender_key_distributions skd
      WHERE skd.sender_member_id = $1
        AND skd.receiver_member_id = cm.id
        AND skd.chain_id >= $2
  )`

	db := r.GetDB(ctx)
	var ids []int64
	if err := sqlx.SelectContext(ctx, db, &ids, query, int64(senderMemberID), latestChainID); err != nil {
		return nil, fmt.Errorf("find pending receivers: %w", err)
	}

	result := make([]chatmember.ID, len(ids))
	for i, id := range ids {
		result[i] = chatmember.ID(id)
	}
	return result, nil
}

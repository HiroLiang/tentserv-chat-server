package e2ee

import (
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
)

func toDistributionDomain(rec *SenderKeyDistributionRecord) *senderkeydistribution.SenderKeyDistribution {
	return &senderkeydistribution.SenderKeyDistribution{
		ID:               senderkeydistribution.ID(rec.ID),
		SenderMemberID:   chatmember.ID(rec.SenderMemberID),
		ReceiverMemberID: chatmember.ID(rec.ReceiverMemberID),
		ChainID:          rec.ChainID,
		DistributedAt:    rec.DistributedAt,
	}
}

func toDistributionRecord(d *senderkeydistribution.SenderKeyDistribution) *SenderKeyDistributionRecord {
	return &SenderKeyDistributionRecord{
		ID:               int64(d.ID),
		SenderMemberID:   int64(d.SenderMemberID),
		ReceiverMemberID: int64(d.ReceiverMemberID),
		ChainID:          d.ChainID,
		DistributedAt:    d.DistributedAt,
	}
}


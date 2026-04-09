package e2ee

import (
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
)

func toDistributionDomain(rec *SenderKeyDistributionRecord) *senderkeydistribution.SenderKeyDistribution {
	return &senderkeydistribution.SenderKeyDistribution{
		ID:                  senderkeydistribution.ID(rec.ID),
		SenderMemberID:      chatmember.ID(rec.SenderMemberID),
		ReceiverMemberID:    chatmember.ID(rec.ReceiverMemberID),
		SenderKeyVersion:    rec.SenderKeyVersion,
		DistributionMessage: rec.DistributionMessage,
		Status:              senderkeydistribution.Status(rec.Status),
		ChainID:             int(rec.SenderKeyVersion),
		DistributedAt:       rec.DistributedAt,
		ConsumedAt:          rec.ConsumedAt,
		FailedAt:            rec.FailedAt,
	}
}

func toDistributionRecord(d *senderkeydistribution.SenderKeyDistribution) *SenderKeyDistributionRecord {
	return &SenderKeyDistributionRecord{
		ID:                  int64(d.ID),
		SenderMemberID:      int64(d.SenderMemberID),
		ReceiverMemberID:    int64(d.ReceiverMemberID),
		SenderKeyVersion:    d.SenderKeyVersion,
		DistributionMessage: d.DistributionMessage,
		Status:              string(d.Status),
		DistributedAt:       d.DistributedAt,
		ConsumedAt:          d.ConsumedAt,
		FailedAt:            d.FailedAt,
	}
}

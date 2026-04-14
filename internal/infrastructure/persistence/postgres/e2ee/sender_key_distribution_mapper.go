package e2ee

import (
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

func toDistributionDomain(rec *SenderKeyDistributionRecord) *senderkeydistribution.SenderKeyDistribution {
	chainID := rec.ChainID
	if chainID < rec.SenderKeyVersion {
		chainID = rec.SenderKeyVersion
	}

	return &senderkeydistribution.SenderKeyDistribution{
		ID:                  senderkeydistribution.ID(rec.ID),
		SenderMemberID:      chatmember.ID(rec.SenderMemberID),
		SenderDeviceID:      shared.DeviceID(parseUUIDOrNil(rec.SenderDeviceID)),
		ReceiverMemberID:    chatmember.ID(rec.ReceiverMemberID),
		ReceiverDeviceID:    shared.DeviceID(parseUUIDOrNil(rec.ReceiverDeviceID)),
		SenderKeyVersion:    rec.SenderKeyVersion,
		DistributionMessage: rec.DistributionMessage,
		Status:              senderkeydistribution.Status(rec.Status),
		ChainID:             chainID,
		DistributedAt:       rec.DistributedAt,
		ConsumedAt:          rec.ConsumedAt,
		FailedAt:            rec.FailedAt,
	}
}

func toDistributionRecord(d *senderkeydistribution.SenderKeyDistribution) *SenderKeyDistributionRecord {
	return &SenderKeyDistributionRecord{
		ID:                  int64(d.ID),
		SenderMemberID:      int64(d.SenderMemberID),
		SenderDeviceID:      d.SenderDeviceID.String(),
		ReceiverMemberID:    int64(d.ReceiverMemberID),
		ReceiverDeviceID:    d.ReceiverDeviceID.String(),
		SenderKeyVersion:    d.SenderKeyVersion,
		ChainID:             d.ChainID,
		DistributionMessage: d.DistributionMessage,
		Status:              string(d.Status),
		DistributedAt:       d.DistributedAt,
		ConsumedAt:          d.ConsumedAt,
		FailedAt:            d.FailedAt,
	}
}

func parseUUIDOrNil(raw string) [16]byte {
	id, err := shared.ParseDeviceID(raw)
	if err != nil {
		return [16]byte{}
	}
	return [16]byte(id)
}

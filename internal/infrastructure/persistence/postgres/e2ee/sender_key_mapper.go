package e2ee

import (
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
)

func toSenderKeyDomain(rec *SenderKeyRecord) (*membersenderkey.MemberSenderKey, error) {
	if len(rec.SenderKeyPublic) != 32 {
		return nil, fmt.Errorf("sender key: invalid public key length %d", len(rec.SenderKeyPublic))
	}

	var pub membersenderkey.SenderKeyPublic
	copy(pub[:], rec.SenderKeyPublic)

	return &membersenderkey.MemberSenderKey{
		ID:                  rec.ID,
		ChatMemberID:        rec.ChatMemberID,
		ChainID:             rec.ChainID,
		SenderKeyPublic:     pub,
		DistributionMessage: rec.DistributionMessage,
		CreatedAt:           rec.CreatedAt,
	}, nil
}

func toSenderKeyRecord(sk *membersenderkey.MemberSenderKey) *SenderKeyRecord {
	return &SenderKeyRecord{
		ID:                  sk.ID,
		ChatMemberID:        sk.ChatMemberID,
		ChainID:             sk.ChainID,
		SenderKeyPublic:     sk.SenderKeyPublic[:],
		DistributionMessage: sk.DistributionMessage,
		CreatedAt:           sk.CreatedAt,
	}
}

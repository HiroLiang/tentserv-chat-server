package e2ee

import (
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
)

func toSenderKeyDomain(rec *SenderKeyRecord) (*membersenderkey.MemberSenderKey, error) {
	chainID := rec.ChainID
	if int64(chainID) < rec.SenderKeyVersion {
		chainID = membersenderkey.ChainID(rec.SenderKeyVersion)
	}

	return &membersenderkey.MemberSenderKey{
		ID:               rec.ID,
		ChatMemberID:     rec.ChatMemberID,
		SenderKeyVersion: rec.SenderKeyVersion,
		KeyFingerprint:   rec.KeyFingerprint,
		ChainID:          chainID,
		CreatedAt:        rec.CreatedAt,
	}, nil
}

func toSenderKeyRecord(sk *membersenderkey.MemberSenderKey) *SenderKeyRecord {
	return &SenderKeyRecord{
		ID:               sk.ID,
		ChatMemberID:     sk.ChatMemberID,
		ChainID:          sk.ChainID,
		SenderKeyVersion: sk.SenderKeyVersion,
		KeyFingerprint:   sk.KeyFingerprint,
		CreatedAt:        sk.CreatedAt,
	}
}

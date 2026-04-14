package e2ee

import (
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

func toSenderKeyDomain(rec *SenderKeyRecord) (*membersenderkey.MemberSenderKey, error) {
	chainID := rec.ChainID
	if int64(chainID) < rec.SenderKeyVersion {
		chainID = membersenderkey.ChainID(rec.SenderKeyVersion)
	}

	return &membersenderkey.MemberSenderKey{
		ID:               rec.ID,
		ChatMemberID:     rec.ChatMemberID,
		SenderDeviceID:   shared.DeviceID(parseUUIDOrNil(rec.SenderDeviceID)),
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
		SenderDeviceID:   sk.SenderDeviceID.String(),
		ChainID:          sk.ChainID,
		SenderKeyVersion: sk.SenderKeyVersion,
		KeyFingerprint:   sk.KeyFingerprint,
		CreatedAt:        sk.CreatedAt,
	}
}

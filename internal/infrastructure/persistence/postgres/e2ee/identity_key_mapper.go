package e2ee

import (
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/useridentitykey"
)

func toIdentityKeyDomain(rec *IdentityKeyRecord) (*useridentitykey.UserIdentityKey, error) {
	if len(rec.PublicKey) != 32 {
		return nil, fmt.Errorf("identity key: invalid public key length %d", len(rec.PublicKey))
	}
	var pub useridentitykey.PublicKey
	copy(pub[:], rec.PublicKey)

	return &useridentitykey.UserIdentityKey{
		ID:          rec.ID,
		UserID:      rec.UserID,
		DeviceID:    rec.DeviceID,
		PublicKey:   pub,
		Fingerprint: useridentitykey.Fingerprint(rec.Fingerprint),
		UploadedAt:  rec.UploadedAt,
	}, nil
}

func toIdentityKeyRecord(k *useridentitykey.UserIdentityKey) *IdentityKeyRecord {
	return &IdentityKeyRecord{
		ID:          k.ID,
		UserID:      k.UserID,
		DeviceID:    k.DeviceID,
		PublicKey:   k.PublicKey[:],
		Fingerprint: string(k.Fingerprint),
		UploadedAt:  k.UploadedAt,
	}
}

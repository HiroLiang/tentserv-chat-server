package e2ee

import (
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/usersignedprekey"
)

func toSignedPreKeyDomain(rec *SignedPreKeyRecord) (*usersignedprekey.UserSignedPreKey, error) {
	if len(rec.PublicKey) != 32 {
		return nil, fmt.Errorf("signed prekey: invalid public key length %d", len(rec.PublicKey))
	}
	if len(rec.Signature) != 64 {
		return nil, fmt.Errorf("signed prekey: invalid signature length %d", len(rec.Signature))
	}

	var pub usersignedprekey.PublicKey
	copy(pub[:], rec.PublicKey)

	var sig usersignedprekey.Signature
	copy(sig[:], rec.Signature)

	return &usersignedprekey.UserSignedPreKey{
		ID:        rec.ID,
		UserID:    rec.UserID,
		DeviceID:  rec.DeviceID,
		KeyID:     rec.KeyID,
		PublicKey: pub,
		Signature: sig,
		IsActive:  rec.IsActive,
		CreatedAt: rec.CreatedAt,
		ExpiresAt: rec.ExpiresAt,
	}, nil
}

func toSignedPreKeyRecord(k *usersignedprekey.UserSignedPreKey) *SignedPreKeyRecord {
	return &SignedPreKeyRecord{
		ID:        k.ID,
		UserID:    k.UserID,
		DeviceID:  k.DeviceID,
		KeyID:     k.KeyID,
		PublicKey: k.PublicKey[:],
		Signature: k.Signature[:],
		IsActive:  k.IsActive,
		CreatedAt: k.CreatedAt,
		ExpiresAt: k.ExpiresAt,
	}
}

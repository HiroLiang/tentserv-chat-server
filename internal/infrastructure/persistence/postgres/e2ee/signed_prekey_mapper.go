package e2ee

import (
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/usersignedprekey"
)

func toSignedPreKeyDomain(rec *SignedPreKeyRecord) (*usersignedprekey.UserSignedPreKey, error) {
	if len(rec.PublicKey) != 32 {
		return nil, fmt.Errorf("signed prekey: invalid public key length %d", len(rec.PublicKey))
	}
	if len(rec.Signature) != 64 {
		return nil, fmt.Errorf("signed prekey: invalid signature length %d", len(rec.Signature))
	}

	deviceID, err := shared.ParseDeviceID(rec.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("signed prekey: invalid device id %q: %w", rec.DeviceID, err)
	}

	var pub usersignedprekey.PublicKey
	copy(pub[:], rec.PublicKey)

	var sig usersignedprekey.Signature
	copy(sig[:], rec.Signature)

	return &usersignedprekey.UserSignedPreKey{
		ID:        rec.ID,
		UserID:    rec.UserID,
		DeviceID:  deviceID,
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
		DeviceID:  k.DeviceID.String(),
		KeyID:     k.KeyID,
		PublicKey: k.PublicKey[:],
		Signature: k.Signature[:],
		IsActive:  k.IsActive,
		CreatedAt: k.CreatedAt,
		ExpiresAt: k.ExpiresAt,
	}
}

package e2ee

import (
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userotpprekey"
)

func toOTPPreKeyDomain(rec *OTPPreKeyRecord) (*userotpprekey.UserOTPPreKey, error) {
	if len(rec.PublicKey) != 32 {
		return nil, fmt.Errorf("otp prekey: invalid public key length %d", len(rec.PublicKey))
	}

	var pub userotpprekey.PublicKey
	copy(pub[:], rec.PublicKey)

	return &userotpprekey.UserOTPPreKey{
		ID:         rec.ID,
		UserID:     rec.UserID,
		DeviceID:   rec.DeviceID,
		KeyID:      rec.KeyID,
		PublicKey:  pub,
		UploadedAt: rec.UploadedAt,
	}, nil
}

func toOTPPreKeyRecord(k *userotpprekey.UserOTPPreKey) *OTPPreKeyRecord {
	return &OTPPreKeyRecord{
		ID:         k.ID,
		UserID:     k.UserID,
		DeviceID:   k.DeviceID,
		KeyID:      k.KeyID,
		PublicKey:  k.PublicKey[:],
		UploadedAt: k.UploadedAt,
	}
}

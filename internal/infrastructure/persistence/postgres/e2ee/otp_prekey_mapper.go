package e2ee

import (
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userotpprekey"
)

func toOTPPreKeyDomain(rec *OTPPreKeyRecord) (*userotpprekey.UserOTPPreKey, error) {
	if len(rec.PublicKey) != 32 {
		return nil, fmt.Errorf("otp prekey: invalid public key length %d", len(rec.PublicKey))
	}

	deviceID, err := shared.ParseDeviceID(rec.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("otp prekey: invalid device id %q: %w", rec.DeviceID, err)
	}

	var pub userotpprekey.PublicKey
	copy(pub[:], rec.PublicKey)

	return &userotpprekey.UserOTPPreKey{
		ID:         rec.ID,
		UserID:     rec.UserID,
		DeviceID:   deviceID,
		KeyID:      rec.KeyID,
		PublicKey:  pub,
		UploadedAt: rec.UploadedAt,
	}, nil
}

func toOTPPreKeyRecord(k *userotpprekey.UserOTPPreKey) *OTPPreKeyRecord {
	return &OTPPreKeyRecord{
		ID:         k.ID,
		UserID:     k.UserID,
		DeviceID:   k.DeviceID.String(),
		KeyID:      k.KeyID,
		PublicKey:  k.PublicKey[:],
		UploadedAt: k.UploadedAt,
	}
}

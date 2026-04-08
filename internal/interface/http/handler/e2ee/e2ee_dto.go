package e2ee

// Identity Key
type UploadIdentityKeyRequest struct {
	DeviceID      string `json:"device_id"       binding:"required"`
	PublicKey     string `json:"public_key"      binding:"required"`
	SignPublicKey string `json:"sign_public_key" binding:"required"`
}

type UploadIdentityKeyResponse struct {
	Fingerprint string `json:"fingerprint"`
}

// Signed Pre-Key
type UploadSignedPreKeyRequest struct {
	DeviceID  string `json:"device_id" binding:"required"`
	KeyID     uint32 `json:"key_id" binding:"required"`
	PublicKey string `json:"public_key" binding:"required"`
	Signature string `json:"signature" binding:"required"`
}

// OTP Pre-Keys
type OTPPreKeyItemRequest struct {
	KeyID     uint32 `json:"key_id" binding:"required"`
	PublicKey string `json:"public_key" binding:"required"`
}

type UploadOTPPreKeysRequest struct {
	DeviceID string                 `json:"device_id" binding:"required"`
	Keys     []OTPPreKeyItemRequest `json:"keys" binding:"required,min=1"`
}

type UploadOTPPreKeysResponse struct {
	Count int `json:"count"`
}

// OTP Count
type CountOTPPreKeysResponse struct {
	Count int `json:"count"`
}

// Key Bundle
type KeyBundleResponse struct {
	IdentityKey     string  `json:"identity_key"`
	IdentityKeySign string  `json:"identity_key_sign"`
	SignedPreKey    string  `json:"signed_pre_key"`
	SPKSignature    string  `json:"spk_signature"`
	SPKKeyID        uint32  `json:"spk_key_id"`
	OTPPreKey       *string `json:"otp_pre_key,omitempty"`
	OTPPreKeyID     *uint32 `json:"otp_pre_key_id,omitempty"`
}

// Key Status (non-consuming existence check)
type CheckKeyStatusResponse struct {
	IdentityKeyExists  bool `json:"identity_key_exists"`
	SignedPreKeyExists bool `json:"signed_pre_key_exists"`
}

// Sender Key (used for both direct and group rooms)
type UploadSenderKeyRequest struct {
	RoomID              int64  `json:"room_id" binding:"required"`
	SenderKeyPublic     string `json:"sender_key_public" binding:"required"`
	DistributionMessage string `json:"distribution_message" binding:"required"`
}

type SenderKeyItemResponse struct {
	ChatMemberID        int64  `json:"chat_member_id"`
	SenderKeyPublic     string `json:"sender_key_public"`
	DistributionMessage string `json:"distribution_message"`
}

type GetSenderKeysResponse struct {
	Keys []SenderKeyItemResponse `json:"keys"`
}

// Sender Key Request
type CreateSenderKeyRequestRequest struct {
	RoomID           int64 `json:"room_id"            binding:"required"`
	ProviderMemberID int64 `json:"provider_member_id" binding:"required"`
}

// Sender Key Distribution Status
type GetSenderKeyDistributionStatusResponse struct {
	// PendingReceivers: member IDs in the room who have not yet fetched my latest sender key.
	PendingReceivers []int64 `json:"pending_receivers"`
	// PendingFromMembers: member IDs whose latest sender key I have not yet fetched.
	PendingFromMembers []int64 `json:"pending_from_members"`
}

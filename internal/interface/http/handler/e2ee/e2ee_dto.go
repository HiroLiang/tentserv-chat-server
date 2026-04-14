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
	IdentityKeyExists  bool   `json:"identity_key_exists"`
	SignedPreKeyExists bool   `json:"signed_pre_key_exists"`
	DeviceID           string `json:"device_id,omitempty"`
	IdentityKey        string `json:"identity_key,omitempty"`
	IdentityKeySign    string `json:"identity_key_sign,omitempty"`
	SignedPreKey       string `json:"signed_pre_key,omitempty"`
	SPKSignature       string `json:"spk_signature,omitempty"`
	SPKKeyID           uint32 `json:"spk_key_id,omitempty"`
	OTPPreKeyCount     int    `json:"otp_prekey_count"`
}

type GetKeyPolicyResponse struct {
	OTPPreKeyTargetCount        int `json:"otp_prekey_target_count"`
	OTPPreKeyReplenishThreshold int `json:"otp_prekey_replenish_threshold"`
}

// Sender Key (used for both direct and group rooms)
type UploadSenderKeyRequest struct {
	RoomID              int64  `json:"room_id" binding:"required"`
	SenderMemberID      int64  `json:"sender_member_id" binding:"required"`
	ReceiverUserID      int64  `json:"receiver_user_id" binding:"required"`
	ReceiverDeviceID    string `json:"receiver_device_id"`
	SenderKeyVersion    int64  `json:"sender_key_version" binding:"required"`
	DistributionMessage string `json:"distribution_message" binding:"required"`
}

type SenderKeyItemResponse struct {
	ChatMemberID     int64  `json:"chat_member_id"`
	ProviderDeviceID string `json:"provider_device_id"`
	SenderKeyVersion int64  `json:"sender_key_version"`
}

type GetSenderKeysResponse struct {
	Keys []SenderKeyItemResponse `json:"keys"`
}

// Sender Key Request
type CreateSenderKeyRequestRequest struct {
	RoomID            int64  `json:"room_id"             binding:"required"`
	ProviderUserID    int64  `json:"provider_user_id"    binding:"required"`
	ProviderDeviceID  string `json:"provider_device_id"  binding:"required"`
	SenderMemberID    int64  `json:"sender_member_id"    binding:"required"`
	RequesterDeviceID string `json:"requester_device_id"`
}

type SenderKeyRouteRefResponse struct {
	UserID   int64  `json:"user_id"`
	MemberID int64  `json:"member_id"`
	DeviceID string `json:"device_id"`
}

// Sender Key Distribution Status
type GetSenderKeyDistributionStatusResponse struct {
	OwnMemberSenderKeyExists bool                        `json:"own_member_sender_key_exists"`
	RequestableSources       []SenderKeyRouteRefResponse `json:"requestable_sources"`
	AvailableFromSources     []SenderKeyRouteRefResponse `json:"available_from_sources"`
	AvailableToTargets       []SenderKeyRouteRefResponse `json:"available_to_targets"`
	PendingReceivers         []SenderKeyRouteRefResponse `json:"pending_receivers"`
	PendingFromSources       []SenderKeyRouteRefResponse `json:"pending_from_sources"`
}

type PendingSenderKeyDistributionItemResponse struct {
	DistributionID      int64  `json:"distribution_id"`
	SenderMemberID      int64  `json:"sender_member_id"`
	SenderDeviceID      string `json:"sender_device_id"`
	ReceiverMemberID    int64  `json:"receiver_member_id"`
	ReceiverDeviceID    string `json:"receiver_device_id"`
	SenderKeyVersion    int64  `json:"sender_key_version"`
	DistributionMessage string `json:"distribution_message"`
}

type GetPendingSenderKeyDistributionsResponse struct {
	Distributions []PendingSenderKeyDistributionItemResponse `json:"distributions"`
}

type ConsumeSenderKeyDistributionRequest struct {
	Status string `json:"status" binding:"required,oneof=consumed failed"`
}

type SelfSenderKeySyncDistributionItemRequest struct {
	SenderMemberID      int64  `json:"sender_member_id" binding:"required"`
	SenderDeviceID      string `json:"sender_device_id" binding:"required"`
	SenderKeyVersion    int64  `json:"sender_key_version" binding:"required"`
	DistributionMessage string `json:"distribution_message" binding:"required"`
}

type BulkSelfSenderKeySyncDistributionsRequest struct {
	Items []SelfSenderKeySyncDistributionItemRequest `json:"items" binding:"required,min=1"`
}

type BulkSelfSenderKeySyncDistributionsResponse struct {
	Count int `json:"count"`
}

type PendingSelfSenderKeySyncDistributionItemResponse struct {
	DistributionID      int64  `json:"distribution_id"`
	SenderMemberID      int64  `json:"sender_member_id"`
	SenderDeviceID      string `json:"sender_device_id"`
	SenderKeyVersion    int64  `json:"sender_key_version"`
	DistributionMessage string `json:"distribution_message"`
}

type GetPendingSelfSenderKeySyncDistributionsResponse struct {
	Distributions []PendingSelfSenderKeySyncDistributionItemResponse `json:"distributions"`
}

type ConsumeSelfSenderKeySyncDistributionRequest struct {
	Status string `json:"status" binding:"required,oneof=consumed failed"`
}

type GetSelfSenderKeySyncResponse struct {
	Exists                 bool                 `json:"exists"`
	Status                 string               `json:"status"`
	RequesterDevice        *SelfSenderKeyDevice `json:"requester_device,omitempty"`
	ProviderDevice         *SelfSenderKeyDevice `json:"provider_device,omitempty"`
	RequesterCurrentDevice bool                 `json:"requester_current_device"`
	ProviderCurrentDevice  bool                 `json:"provider_current_device"`
	LastError              string               `json:"last_error,omitempty"`
	RequestedAtMS          int64                `json:"requested_at_ms,omitempty"`
	ProviderClaimedAtMS    int64                `json:"provider_claimed_at_ms,omitempty"`
	UploadedAtMS           int64                `json:"uploaded_at_ms,omitempty"`
	CompletedAtMS          int64                `json:"completed_at_ms,omitempty"`
	FailedAtMS             int64                `json:"failed_at_ms,omitempty"`
}

type SelfSenderKeyDevice struct {
	DeviceID      string `json:"device_id"`
	DeviceName    string `json:"device_name"`
	Platform      string `json:"platform"`
	LastIP        string `json:"last_ip,omitempty"`
	BindingStatus string `json:"binding_status,omitempty"`
}

type FailSelfSenderKeySyncRequest struct {
	LastError string `json:"last_error"`
	Retryable bool   `json:"retryable"`
}

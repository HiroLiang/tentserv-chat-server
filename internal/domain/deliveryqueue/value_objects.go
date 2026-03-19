package deliveryqueue

type ID int64

type Status string

const (
	StatusPending   Status = "pending"
	StatusDelivered Status = "delivered"
)

type PayloadType string

const (
	PayloadTypeSenderKey    PayloadType = "sender_key"
	PayloadTypeMessage      PayloadType = "message"
	PayloadTypeSPKUpdate    PayloadType = "spk_update"
	PayloadTypeReplenishOTP PayloadType = "e2ee.replenish_otp_keys"
)

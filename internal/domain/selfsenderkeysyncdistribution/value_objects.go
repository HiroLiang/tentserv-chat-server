package selfsenderkeysyncdistribution

type Status string

const (
	StatusAvailable Status = "available"
	StatusConsumed  Status = "consumed"
	StatusFailed    Status = "failed"
)

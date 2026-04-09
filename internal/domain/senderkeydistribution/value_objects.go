package senderkeydistribution

type ID int64

type Status string

const (
	StatusAvailable Status = "available"
	StatusConsumed  Status = "consumed"
	StatusFailed    Status = "failed"
)

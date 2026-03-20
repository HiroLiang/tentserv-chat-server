package friendship

type Status string

const (
	StatusPending  Status = "pending"
	StatusAccepted Status = "accepted"
	StatusBlocked  Status = "blocked"
)

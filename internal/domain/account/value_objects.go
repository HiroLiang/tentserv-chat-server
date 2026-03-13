package account

type Status string

const (
	Active   Status = "active"
	Inactive Status = "inactive"
	Banned   Status = "banned"
	Applying Status = "applying"
	Deleted  Status = "deleted"
)

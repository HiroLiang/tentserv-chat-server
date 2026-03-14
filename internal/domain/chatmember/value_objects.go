package chatmember

type ID int64

type Role string

const (
	Owner  Role = "owner"
	Admin  Role = "admin"
	Member Role = "member"
	Guest  Role = "guest"
)

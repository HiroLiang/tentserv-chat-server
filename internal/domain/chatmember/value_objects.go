package chatmember

type ID int64

type Role string

const (
	Owner  Role = "owner"
	Admin  Role = "admin"
	Member Role = "member"
	Guest  Role = "guest"
)

// CanSendMessage returns true for roles that are permitted to post messages.
// Guest is read-only.
func CanSendMessage(role Role) bool {
	return role == Owner || role == Admin || role == Member
}

// CanUploadMedia returns true for roles that are permitted to upload files/images.
// Guest is read-only.
func CanUploadMedia(role Role) bool {
	return role == Owner || role == Admin || role == Member
}

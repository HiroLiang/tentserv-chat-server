package friendship

import "errors"

var (
	ErrFriendshipNotFound      = errors.New("friendship not found")
	ErrAlreadyFriends          = errors.New("already friends")
	ErrFriendshipAlreadyExists = errors.New("friendship already exists")
	ErrFriendshipNotPending    = errors.New("friendship is not pending")
	ErrForbidden               = errors.New("forbidden")
)

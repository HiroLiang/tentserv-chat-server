package friendship

import "errors"

var (
	ErrFriendshipNotFound = errors.New("friendship not found")
	ErrAlreadyFriends     = errors.New("already friends")
)

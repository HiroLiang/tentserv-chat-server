package friendship

import "errors"

var (
	ErrFriendshipNotFound      = errors.New("friendship not found")
	ErrAlreadyFriends          = errors.New("already friends")
	ErrFriendshipAlreadyExists = errors.New("friendship already exists")
	ErrFriendshipNotPending    = errors.New("friendship is not pending")
	ErrSelfFriendship          = errors.New("cannot send a friend request to yourself")
	ErrForbidden               = errors.New("forbidden")
	ErrAlreadyBlocked          = errors.New("user is already blocked")
	ErrNotBlocked              = errors.New("user is not blocked")
)

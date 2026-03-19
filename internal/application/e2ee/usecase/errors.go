package usecase

import "errors"

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrIdentityNotFound = errors.New("identity key not found")
	ErrNotRoomMember    = errors.New("not a room member")
)

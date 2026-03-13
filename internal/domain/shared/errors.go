package shared

import "errors"

var (
	ErrInvalidID    = errors.New("invalid id")
	ErrInvalidEmail = errors.New("invalid email")

	ErrSendingEmail = errors.New("sending email failed")

	ErrInvalidFileName = errors.New("invalid file name")
)

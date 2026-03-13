package account

import "errors"

var (
	ErrEmailExist      = errors.New("email exist")
	ErrAccountExist    = errors.New("account exist")
	ErrAccountNotFound = errors.New("account not found")
)

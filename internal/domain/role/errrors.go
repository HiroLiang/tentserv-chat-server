package role

import "errors"

var (
	ErrInvalidType   = errors.New("invalid role type")
	ErrNotFound      = errors.New("role not found")
	ErrAlreadyExists = errors.New("role already exists")
)

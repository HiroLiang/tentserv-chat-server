package participant

import "errors"

var (
	ErrNotFound    = errors.New("participant not found")
	ErrAlreadyExists = errors.New("participant already exists")
)

package userotpprekey

import "errors"

var (
	ErrNotFound  = errors.New("one-time prekey not found")
	ErrPoolEmpty = errors.New("one-time prekey pool is empty")
)

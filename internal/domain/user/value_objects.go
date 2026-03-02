package user

import (
	"strconv"
)

type ID int64

func ParseID(str string) (ID, error) {
	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, ErrInvalidID
	}
	return ID(i), nil
}

type Email string

func ParseEmail(v string) (Email, error) {
	if len(v) < 5 || !containsAt(v) {
		return "", ErrInvalidEmail
	}
	return Email(v), nil
}

func containsAt(s string) bool {
	for _, c := range s {
		if c == '@' {
			return true
		}
	}
	return false
}

type Status string

const (
	Active   Status = "active"
	Inactive Status = "inactive"
	Banned   Status = "banned"
	Applying Status = "applying"
	Deleted  Status = "deleted"
)

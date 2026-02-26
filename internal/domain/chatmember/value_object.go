package chatmember

import (
	"errors"
	"fmt"
	"strings"
)

type ID int64

type Role string

const (
	Owner  Role = "owner"
	Admin  Role = "admin"
	Member Role = "member"
	Guest  Role = "guest"
)

var ErrInvalidRole = errors.New("invalid role")

func ParseRole(s string) (Role, error) {
	r := Role(strings.ToLower(strings.TrimSpace(s)))
	switch r {
	case Owner, Admin, Member, Guest:
		return r, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidRole, s)
	}
}

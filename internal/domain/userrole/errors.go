package userrole

import "errors"

var (
	ErrRevokeFailed            = errors.New("revoke failed")
	ErrUserRoleAlreadyAssigned = errors.New("user role already assigned")
)

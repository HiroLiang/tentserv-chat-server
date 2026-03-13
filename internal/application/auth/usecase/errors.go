package usecase

import "errors"

var (
	ErrRegisterFailed = errors.New("failed to register account")
	ErrLoginFailed    = errors.New("failed to login")

	ErrInvalidDeviceID = errors.New("invalid device id")
	ErrInvalidPassword = errors.New("invalid password")
	ErrInvalidEmail    = errors.New("invalid email")

	ErrEmailExist = errors.New("email exist")

	ErrAccountExist    = errors.New("account exist")
	ErrAccountNotFound = errors.New("account not found")
	ErrAccountBanned   = errors.New("account is banned")
	ErrAccountApplying = errors.New("account registration is pending approval")
	ErrAccountInactive = errors.New("account is locked, please contact admin to unlock")

	ErrUserNotFound = errors.New("user not found")

	ErrPasswordError = errors.New("password error")
)

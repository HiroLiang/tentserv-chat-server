package auth

import "time"

type AccessToken string

type RefreshToken string

type TokenPair struct {
	AccessToken  AccessToken
	RefreshToken RefreshToken
	ExpiresAt    time.Time
}

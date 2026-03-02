package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
)

type SHA256HMACer struct {
	secret []byte
}

func NewSHA256HMACer(secret string) *SHA256HMACer {
	return &SHA256HMACer{secret: []byte(secret)}
}

func (h *SHA256HMACer) Sign(message string) string {
	mac := hmac.New(sha256.New, h.secret)
	mac.Write([]byte(message))
	sum := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(sum)
}

func (h *SHA256HMACer) Verify(message, provided string) bool {
	expected := h.Sign(message)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}

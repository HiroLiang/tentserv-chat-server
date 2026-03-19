package security

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared/security"
)

type ContentHasher struct{}

var _ security.Hasher = (*ContentHasher)(nil)

func NewContentHasher() *ContentHasher { return &ContentHasher{} }

func (h *ContentHasher) Hash(str string) (string, error) {
	return h.HashBytes([]byte(str))
}

func (h *ContentHasher) HashBytes(bytes []byte) (string, error) {
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:]), nil
}

func (h *ContentHasher) Verify(hashed, plain string) bool {
	if len(hashed) != 64 {
		return false
	}

	plainHash, err := h.Hash(plain)
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(hashed), []byte(plainHash)) == 1
}

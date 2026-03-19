package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"strings"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared/security"
	"golang.org/x/crypto/argon2"
)

type Argon2Hasher struct{}

var _ security.Hasher = (*Argon2Hasher)(nil)

func NewArgon2Hasher() *Argon2Hasher { return &Argon2Hasher{} }

func (h *Argon2Hasher) Hash(password string) (string, error) {
	return h.HashBytes([]byte(password))
}

func (h *Argon2Hasher) HashBytes(password []byte) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := h.getArgon2Hash(password, salt)
	encoded := base64.RawStdEncoding.EncodeToString(hash)
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	return encodedSalt + ":" + encoded, nil
}

func (h *Argon2Hasher) Verify(hashed, plain string) bool {
	parts := strings.Split(plain, ":")
	if len(parts) != 2 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	newHash := h.getArgon2Hash([]byte(hashed), salt)
	return subtle.ConstantTimeCompare(hash, newHash) == 1
}

func (h *Argon2Hasher) getArgon2Hash(bytes []byte, salt []byte) []byte {
	return argon2.IDKey(bytes, salt, 1, 64*1024, 4, 32)
}

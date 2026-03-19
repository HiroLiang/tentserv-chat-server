package crypto

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"

	appcrypto "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/crypto"
)

type E2EEVerifier struct{}

var _ appcrypto.KeyVerifier = (*E2EEVerifier)(nil)

func NewE2EEVerifier() *E2EEVerifier {
	return &E2EEVerifier{}
}

// VerifySignedPreKey verifies that signature is a valid Ed25519 signature of
// publicKey by the private key corresponding to identityKey.
func (v *E2EEVerifier) VerifySignedPreKey(identityKey, publicKey, signature []byte) bool {
	return ed25519.Verify(identityKey, publicKey, signature)
}

// FingerprintKey returns hex(SHA-256(publicKey)).
func (v *E2EEVerifier) FingerprintKey(publicKey []byte) string {
	sum := sha256.Sum256(publicKey)
	return hex.EncodeToString(sum[:])
}

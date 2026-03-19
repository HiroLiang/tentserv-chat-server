package crypto

// KeyVerifier provides cryptographic verification for E2EE key material.
type KeyVerifier interface {

	// VerifySignedPreKey verifies the Ed25519 signature of publicKey by identityKey.
	VerifySignedPreKey(identityKey, publicKey, signature []byte) bool

	// FingerprintKey returns hex(SHA-256(publicKey)).
	FingerprintKey(publicKey []byte) string
}

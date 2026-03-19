package crypto_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	infraCrypto "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/shared/crypto"
)

func TestE2EEVerifier_VerifySignedPreKey(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	message := []byte("test-prekey-public-bytes")
	sig := ed25519.Sign(priv, message)

	v := infraCrypto.NewE2EEVerifier()

	t.Run("valid signature", func(t *testing.T) {
		if !v.VerifySignedPreKey(pub, message, sig) {
			t.Error("expected valid signature to return true")
		}
	})

	t.Run("tampered message", func(t *testing.T) {
		tampered := append([]byte{}, message...)
		tampered[0] ^= 0xFF
		if v.VerifySignedPreKey(pub, tampered, sig) {
			t.Error("expected tampered message to return false")
		}
	})

	t.Run("tampered signature", func(t *testing.T) {
		badSig := append([]byte{}, sig...)
		badSig[0] ^= 0xFF
		if v.VerifySignedPreKey(pub, message, badSig) {
			t.Error("expected tampered signature to return false")
		}
	})

	t.Run("wrong identity key", func(t *testing.T) {
		otherPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate key: %v", err)
		}
		if v.VerifySignedPreKey(otherPub, message, sig) {
			t.Error("expected wrong identity key to return false")
		}
	})
}

func TestE2EEVerifier_FingerprintKey(t *testing.T) {
	v := infraCrypto.NewE2EEVerifier()

	key := []byte("test-key-material-32-bytes------")
	fp1 := v.FingerprintKey(key)
	fp2 := v.FingerprintKey(key)

	if fp1 != fp2 {
		t.Error("fingerprint should be deterministic")
	}

	if len(fp1) != 64 {
		t.Errorf("expected 64 hex chars (SHA-256), got %d", len(fp1))
	}

	other := []byte("different-key-material----------")
	fpOther := v.FingerprintKey(other)
	if fp1 == fpOther {
		t.Error("different keys should produce different fingerprints")
	}
}

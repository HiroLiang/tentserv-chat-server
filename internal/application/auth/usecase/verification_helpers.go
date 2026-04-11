package usecase

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
)

const (
	verificationCodeLength   = 6
	verificationMaxAttempts  = 3
)

type verificationAttemptError struct {
	remainingAttempts int
	cause             error
}

func (e *verificationAttemptError) Error() string {
	return e.cause.Error()
}

func (e *verificationAttemptError) Unwrap() error {
	return e.cause
}

func (e *verificationAttemptError) RemainingAttempts() int {
	return e.remainingAttempts
}

func newVerificationAttemptError(cause error, remainingAttempts int) error {
	return &verificationAttemptError{
		remainingAttempts: remainingAttempts,
		cause:             cause,
	}
}

func VerificationAttemptsRemaining(err error) (int, bool) {
	attemptErr, ok := err.(*verificationAttemptError)
	if !ok {
		return 0, false
	}
	return attemptErr.RemainingAttempts(), true
}

func generateVerificationToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateVerificationCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", verificationCodeLength, n.Int64()), nil
}

func newVerificationSession(accountID int64, ttl time.Duration) (string, port.VerificationSession, error) {
	token, err := generateVerificationToken()
	if err != nil {
		return "", port.VerificationSession{}, err
	}

	code, err := generateVerificationCode()
	if err != nil {
		return "", port.VerificationSession{}, err
	}

	expiresAt := time.Now().UTC().Add(ttl)
	return token, port.VerificationSession{
		AccountID:         accountID,
		Code:              code,
		ExpiresAtMS:       expiresAt.UnixMilli(),
		RemainingAttempts: verificationMaxAttempts,
	}, nil
}

func verificationSessionTTL(session port.VerificationSession) time.Duration {
	return time.Until(time.UnixMilli(session.ExpiresAtMS))
}

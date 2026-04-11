package verification

import (
	"context"
	"testing"
	"time"

	authPort "github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
)

type verificationCacheStub struct {
	values      map[string][]byte
	ttls        map[string]time.Duration
	deleteCalls []string
}

func newVerificationCacheStub() *verificationCacheStub {
	return &verificationCacheStub{
		values: map[string][]byte{},
		ttls:   map[string]time.Duration{},
	}
}

func (s *verificationCacheStub) Get(_ context.Context, key string) ([]byte, bool, error) {
	value, ok := s.values[key]
	return value, ok, nil
}

func (s *verificationCacheStub) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	s.values[key] = append([]byte(nil), value...)
	s.ttls[key] = ttl
	return nil
}

func (s *verificationCacheStub) Delete(_ context.Context, key string) error {
	s.deleteCalls = append(s.deleteCalls, key)
	delete(s.values, key)
	delete(s.ttls, key)
	return nil
}

func (s *verificationCacheStub) DeleteByPrefix(context.Context, string) error {
	return nil
}

func TestVerificationStore_StoreUsesCacheTTLWhenRetentionIsLonger(t *testing.T) {
	cache := newVerificationCacheStub()
	store := NewVerificationStore(cache, 24*time.Hour)
	session := authPort.VerificationSession{
		AccountID:         101,
		Code:              "123456",
		ExpiresAtMS:       time.Now().Add(3 * time.Minute).UnixMilli(),
		RemainingAttempts: 3,
	}

	if err := store.Store(context.Background(), "active-token", session, 3*time.Minute); err != nil {
		t.Fatalf("store session: %v", err)
	}

	if got := cache.ttls[tokenCacheKey("active-token")]; got != 24*time.Hour {
		t.Fatalf("expected token cache ttl=24h, got %s", got)
	}
	if got := cache.ttls[accountCacheKey(session.AccountID)]; got != 24*time.Hour {
		t.Fatalf("expected account cache ttl=24h, got %s", got)
	}
}

func TestVerificationStore_GetReturnsExpiredSessionWhileRetained(t *testing.T) {
	cache := newVerificationCacheStub()
	store := NewVerificationStore(cache, 24*time.Hour)
	session := authPort.VerificationSession{
		AccountID:         101,
		Code:              "123456",
		ExpiresAtMS:       time.Now().Add(-time.Minute).UnixMilli(),
		RemainingAttempts: 3,
	}

	if err := store.Store(context.Background(), "expired-token", session, time.Minute); err != nil {
		t.Fatalf("store session: %v", err)
	}

	got, ok, err := store.Get(context.Background(), "expired-token")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if !ok {
		t.Fatalf("expected retained expired session to still be retrievable")
	}
	if got.ExpiresAtMS != session.ExpiresAtMS {
		t.Fatalf("expected expires_at_ms=%d, got %d", session.ExpiresAtMS, got.ExpiresAtMS)
	}
}

func TestVerificationStore_FindTokenByAccountIDTreatsExpiredSessionAsMissing(t *testing.T) {
	cache := newVerificationCacheStub()
	store := NewVerificationStore(cache, 24*time.Hour)
	session := authPort.VerificationSession{
		AccountID:         101,
		Code:              "123456",
		ExpiresAtMS:       time.Now().Add(-time.Minute).UnixMilli(),
		RemainingAttempts: 3,
	}

	if err := store.Store(context.Background(), "expired-token", session, time.Minute); err != nil {
		t.Fatalf("store session: %v", err)
	}

	token, ok, err := store.FindTokenByAccountID(context.Background(), session.AccountID)
	if err != nil {
		t.Fatalf("find token by account: %v", err)
	}
	if ok || token != "" {
		t.Fatalf("expected expired session lookup to be treated as missing, got token=%q ok=%t", token, ok)
	}
	if _, ok := cache.values[tokenCacheKey("expired-token")]; ok {
		t.Fatalf("expected expired token payload to be cleaned up")
	}
	if _, ok := cache.values[accountCacheKey(session.AccountID)]; ok {
		t.Fatalf("expected expired account mapping to be cleaned up")
	}
}

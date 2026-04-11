package verification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/cache"
)

const (
	tokenKeyPrefix   = "email_verify:token:"
	accountKeyPrefix = "email_verify:account:"
)

type VerificationStore struct {
	cache    cache.Cache
	cacheTTL time.Duration
}

func NewVerificationStore(c cache.Cache, cacheTTL time.Duration) *VerificationStore {
	return &VerificationStore{cache: c, cacheTTL: cacheTTL}
}

var _ port.VerificationStore = (*VerificationStore)(nil)

func (s *VerificationStore) Store(ctx context.Context, token string, session port.VerificationSession, ttl time.Duration) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	storeTTL := s.storeTTL(ttl)
	if err := s.cache.Set(ctx, tokenCacheKey(token), payload, storeTTL); err != nil {
		return err
	}
	return s.cache.Set(ctx, accountCacheKey(session.AccountID), []byte(token), storeTTL)
}

func (s *VerificationStore) Get(ctx context.Context, token string) (port.VerificationSession, bool, error) {
	data, ok, err := s.cache.Get(ctx, tokenCacheKey(token))
	if err != nil {
		return port.VerificationSession{}, false, err
	}
	if !ok {
		return port.VerificationSession{}, false, nil
	}

	var session port.VerificationSession
	if err := json.Unmarshal(data, &session); err != nil {
		return port.VerificationSession{}, false, err
	}
	return session, true, nil
}

func (s *VerificationStore) FindTokenByAccountID(ctx context.Context, accountID int64) (string, bool, error) {
	data, ok, err := s.cache.Get(ctx, accountCacheKey(accountID))
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}

	token := string(data)
	session, sessionOK, err := s.Get(ctx, token)
	if err != nil {
		return "", false, err
	}
	if !sessionOK {
		if err := s.cache.Delete(ctx, accountCacheKey(accountID)); err != nil {
			return "", false, err
		}
		return "", false, nil
	}
	if isSessionExpired(session, time.Now().UTC()) {
		if err := s.cache.Delete(ctx, accountCacheKey(accountID)); err != nil {
			return "", false, err
		}
		if err := s.cache.Delete(ctx, tokenCacheKey(token)); err != nil {
			return "", false, err
		}
		return "", false, nil
	}

	return token, true, nil
}

func (s *VerificationStore) Delete(ctx context.Context, token string) error {
	session, ok, err := s.Get(ctx, token)
	if err != nil {
		return err
	}
	if ok {
		if err := s.cache.Delete(ctx, accountCacheKey(session.AccountID)); err != nil {
			return err
		}
	}
	return s.cache.Delete(ctx, tokenCacheKey(token))
}

func tokenCacheKey(token string) string {
	return tokenKeyPrefix + token
}

func accountCacheKey(accountID int64) string {
	return accountKeyPrefix + fmt.Sprintf("%d", accountID)
}

func (s *VerificationStore) storeTTL(sessionTTL time.Duration) time.Duration {
	if s.cacheTTL > sessionTTL {
		return s.cacheTTL
	}
	return sessionTTL
}

func isSessionExpired(session port.VerificationSession, now time.Time) bool {
	return session.ExpiresAtMS <= now.UnixMilli()
}

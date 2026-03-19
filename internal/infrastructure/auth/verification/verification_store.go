package verification

import (
	"context"
	"strconv"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/cache"
)

const keyPrefix = "email_verify:"

type VerificationStore struct {
	cache cache.Cache
}

func NewVerificationStore(c cache.Cache) *VerificationStore {
	return &VerificationStore{cache: c}
}

var _ port.VerificationStore = (*VerificationStore)(nil)

func (s *VerificationStore) Store(ctx context.Context, token string, accountID int64, ttl time.Duration) error {
	return s.cache.Set(ctx, keyPrefix+token, []byte(strconv.FormatInt(accountID, 10)), ttl)
}

func (s *VerificationStore) Get(ctx context.Context, token string) (int64, bool, error) {
	data, ok, err := s.cache.Get(ctx, keyPrefix+token)
	if err != nil {
		return 0, false, err
	}
	if !ok {
		return 0, false, nil
	}
	id, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func (s *VerificationStore) Delete(ctx context.Context, token string) error {
	return s.cache.Delete(ctx, keyPrefix+token)
}

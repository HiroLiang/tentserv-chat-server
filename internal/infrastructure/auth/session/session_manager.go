package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/HiroLiang/goat-server/internal/application/auth/port"
	"github.com/HiroLiang/goat-server/internal/domain/auth"
	"github.com/HiroLiang/goat-server/internal/domain/cache"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
	postgresSession "github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/session"
)

const (
	accessKeyPrefix = "session:access:"
	idKeyPrefix     = "session:id:"
)

type SessionManager struct {
	cache       cache.Cache
	sessionRepo *postgresSession.SessionRepository
	expiration  time.Duration
}

func NewSessionManager(
	cache cache.Cache,
	sessionRepo *postgresSession.SessionRepository,
	expiration time.Duration,
) *SessionManager {
	return &SessionManager{
		cache:       cache,
		sessionRepo: sessionRepo,
		expiration:  expiration,
	}
}

var _ port.SessionManager = (*SessionManager)(nil)

// Create generates a new token pair, persists to Postgres, and caches in Redis.
func (s *SessionManager) Create(ctx context.Context, input auth.CreateSessionInput) (auth.TokenPair, error) {
	rawAccess, err := generateSecureToken(32)
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("%w: %v", auth.ErrGenerateToken, err)
	}
	rawRefresh, err := generateSecureToken(32)
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("%w: %v", auth.ErrGenerateToken, err)
	}

	accessHash := hashToken(rawAccess)
	refreshHash := hashToken(rawRefresh)
	expiresAt := time.Now().Add(s.expiration)

	sessionID, err := s.sessionRepo.Create(ctx,
		int64(input.AccountID),
		int64(input.UserID),
		input.DeviceID.String(),
		refreshHash,
		expiresAt,
	)
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("%w: %v", auth.ErrGenerateToken, err)
	}

	tokenPair := auth.TokenPair{
		AccessToken:  auth.AccessToken(rawAccess),
		RefreshToken: auth.RefreshToken(rawRefresh),
		ExpiresAt:    expiresAt,
	}

	session := &auth.Session{
		ID:        strconv.FormatInt(sessionID, 10),
		AccountID: input.AccountID,
		UserID:    input.UserID,
		DeviceID:  input.DeviceID,
		Token:     tokenPair,
		CreatedAt: time.Now(),
	}

	if err := s.setCacheEntries(ctx, sessionID, accessHash, session); err != nil {
		return auth.TokenPair{}, fmt.Errorf("%w: %v", auth.ErrGenerateToken, err)
	}

	return tokenPair, nil
}

// FindByToken looks up the session by access token hash (Redis primary, DB not used for access tokens).
// Slides the expiry window on every call.
func (s *SessionManager) FindByToken(ctx context.Context, token auth.AccessToken) (*auth.Session, error) {
	hash := hashToken(string(token))

	data, ok, err := s.cache.Get(ctx, accessKey(hash))
	if err != nil || !ok {
		return nil, auth.ErrSessionNotFound
	}

	var session auth.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, auth.ErrSessionNotFound
	}

	sessionID, err := strconv.ParseInt(session.ID, 10, 64)
	if err != nil {
		return nil, auth.ErrSessionNotFound
	}

	// Extend TTL (sliding window)
	newExpiry := time.Now().Add(s.expiration)
	session.Token.ExpiresAt = newExpiry

	if err := s.setCacheEntries(ctx, sessionID, hash, &session); err != nil {
		return nil, auth.ErrSessionNotFound
	}

	// Async: update DB last_used_at and expires_at
	go func() {
		bgCtx := context.Background()
		_ = s.sessionRepo.UpdateExpiresAt(bgCtx, sessionID, newExpiry)
	}()

	return &session, nil
}

// Refresh issues a new token pair given a valid refresh token.
// Refresh tokens are validated against Postgres; old Redis entries are cleaned up.
func (s *SessionManager) Refresh(ctx context.Context, token auth.RefreshToken) (auth.TokenPair, error) {
	hash := hashToken(string(token))

	rec, err := s.sessionRepo.FindByRefreshTokenHash(ctx, hash)
	if err != nil {
		return auth.TokenPair{}, auth.ErrRefreshToken
	}
	if rec.Revoked || rec.ExpiresAt.Before(time.Now()) {
		return auth.TokenPair{}, auth.ErrRefreshToken
	}

	// Clean up old Redis entries if they still exist
	if oldHashBytes, ok, _ := s.cache.Get(ctx, idKey(rec.ID)); ok {
		_ = s.cache.Delete(ctx, accessKey(string(oldHashBytes)))
		_ = s.cache.Delete(ctx, idKey(rec.ID))
	}

	rawAccess, err := generateSecureToken(32)
	if err != nil {
		return auth.TokenPair{}, auth.ErrRefreshToken
	}
	rawRefresh, err := generateSecureToken(32)
	if err != nil {
		return auth.TokenPair{}, auth.ErrRefreshToken
	}

	newAccessHash := hashToken(rawAccess)
	newRefreshHash := hashToken(rawRefresh)
	expiresAt := time.Now().Add(s.expiration)

	if err := s.sessionRepo.UpdateRefreshToken(ctx, rec.ID, newRefreshHash, expiresAt); err != nil {
		return auth.TokenPair{}, auth.ErrRefreshToken
	}

	deviceID, err := shared.ParseDeviceID(rec.DeviceID)
	if err != nil {
		return auth.TokenPair{}, auth.ErrRefreshToken
	}

	tokenPair := auth.TokenPair{
		AccessToken:  auth.AccessToken(rawAccess),
		RefreshToken: auth.RefreshToken(rawRefresh),
		ExpiresAt:    expiresAt,
	}

	session := &auth.Session{
		ID:        strconv.FormatInt(rec.ID, 10),
		AccountID: shared.AccountID(rec.AccountID),
		UserID:    shared.UserID(rec.UserID),
		DeviceID:  deviceID,
		Token:     tokenPair,
		CreatedAt: rec.CreatedAt,
	}
	_ = s.setCacheEntries(ctx, rec.ID, newAccessHash, session)

	return tokenPair, nil
}

// Revoke deletes the session from Redis and marks it revoked in Postgres.
func (s *SessionManager) Revoke(ctx context.Context, token auth.AccessToken) error {
	hash := hashToken(string(token))

	data, ok, _ := s.cache.Get(ctx, accessKey(hash))
	if !ok {
		return nil // already expired or revoked
	}

	var session auth.Session
	if err := json.Unmarshal(data, &session); err == nil {
		if sessionID, err := strconv.ParseInt(session.ID, 10, 64); err == nil {
			_ = s.cache.Delete(ctx, idKey(sessionID))
			_ = s.sessionRepo.Revoke(ctx, sessionID)
		}
	}

	return s.cache.Delete(ctx, accessKey(hash))
}

// RevokeAllForUser revokes all sessions belonging to an account.
func (s *SessionManager) RevokeAllForUser(ctx context.Context, accountID shared.AccountID) error {
	ids, err := s.sessionRepo.FindActiveIDsByAccountID(ctx, int64(accountID))
	if err != nil {
		return err
	}
	for _, id := range ids {
		s.deleteBySessionID(ctx, id)
	}
	return s.sessionRepo.RevokeAllForAccount(ctx, int64(accountID))
}

// RevokeAll revokes every active session.
func (s *SessionManager) RevokeAll(ctx context.Context) error {
	ids, err := s.sessionRepo.FindAllActiveIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		s.deleteBySessionID(ctx, id)
	}
	return s.sessionRepo.RevokeAll(ctx)
}

// SwitchUser updates the UserID on the cached session and in Postgres.
func (s *SessionManager) SwitchUser(ctx context.Context, token auth.AccessToken, userID shared.UserID) error {
	hash := hashToken(string(token))

	data, ok, err := s.cache.Get(ctx, accessKey(hash))
	if err != nil || !ok {
		return auth.ErrSessionNotFound
	}

	var session auth.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return auth.ErrSessionNotFound
	}

	sessionID, err := strconv.ParseInt(session.ID, 10, 64)
	if err != nil {
		return auth.ErrSessionNotFound
	}

	if err := s.sessionRepo.UpdateUserID(ctx, sessionID, int64(userID)); err != nil {
		return auth.ErrSessionNotFound
	}

	session.UserID = userID
	return s.setCacheEntries(ctx, sessionID, hash, &session)
}

// --- helpers ---

// setCacheEntries stores session JSON and the id→accessHash reverse mapping.
func (s *SessionManager) setCacheEntries(ctx context.Context, sessionID int64, accessHash string, session *auth.Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	if err := s.cache.Set(ctx, accessKey(accessHash), data, s.expiration); err != nil {
		return err
	}
	return s.cache.Set(ctx, idKey(sessionID), []byte(accessHash), s.expiration)
}

// deleteBySessionID removes both Redis keys for a given session ID.
func (s *SessionManager) deleteBySessionID(ctx context.Context, sessionID int64) {
	if hashBytes, ok, _ := s.cache.Get(ctx, idKey(sessionID)); ok {
		_ = s.cache.Delete(ctx, accessKey(string(hashBytes)))
	}
	_ = s.cache.Delete(ctx, idKey(sessionID))
}

func accessKey(hash string) string {
	return accessKeyPrefix + hash
}

func idKey(sessionID int64) string {
	return idKeyPrefix + strconv.FormatInt(sessionID, 10)
}

func generateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

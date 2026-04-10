package e2ee

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userotpprekey"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOTPPreKeyRepository_ConsumeOneDeletesWithSkipLocked(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewOTPPreKeyRepository(sqlx.NewDb(db, "postgres"))
	deviceID := mustOTPRepoDeviceID(t)
	publicKey := otpRepoBytesOf(9, 32)

	mock.ExpectQuery(`(?s)WITH victim AS .*FOR UPDATE SKIP LOCKED.*DELETE FROM public\.user_one_time_pre_keys.*RETURNING public\.user_one_time_pre_keys\.id,\s+public\.user_one_time_pre_keys\.user_id,\s+public\.user_one_time_pre_keys\.device_id,\s+public\.user_one_time_pre_keys\.key_id,\s+public\.user_one_time_pre_keys\.public_key,\s+public\.user_one_time_pre_keys\.uploaded_at`).
		WithArgs(shared.UserID(42), deviceID.String()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "device_id", "key_id", "public_key", "uploaded_at"}).
			AddRow(1, 42, deviceID.String(), 7, publicKey, time.Now()))

	got, err := repo.ConsumeOne(context.Background(), 42, deviceID)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, userotpprekey.KeyID(7), got.KeyID)
	assert.Equal(t, publicKey, got.PublicKey[:])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOTPPreKeyRepository_ConsumeOneEmptyPoolMapsDomainError(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewOTPPreKeyRepository(sqlx.NewDb(db, "postgres"))
	deviceID := mustOTPRepoDeviceID(t)

	mock.ExpectQuery(`(?s)WITH victim AS .*FOR UPDATE SKIP LOCKED.*DELETE FROM public\.user_one_time_pre_keys.*RETURNING public\.user_one_time_pre_keys\.id,\s+public\.user_one_time_pre_keys\.user_id,\s+public\.user_one_time_pre_keys\.device_id,\s+public\.user_one_time_pre_keys\.key_id,\s+public\.user_one_time_pre_keys\.public_key,\s+public\.user_one_time_pre_keys\.uploaded_at`).
		WithArgs(shared.UserID(42), deviceID.String()).
		WillReturnError(sql.ErrNoRows)

	got, err := repo.ConsumeOne(context.Background(), 42, deviceID)

	require.Error(t, err)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, userotpprekey.ErrPoolEmpty)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOTPPreKeyRepository_AddBatchUsesOnConflictDoNothing(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewOTPPreKeyRepository(sqlx.NewDb(db, "postgres"))
	deviceID := mustOTPRepoDeviceID(t)
	var publicKey userotpprekey.PublicKey
	copy(publicKey[:], otpRepoBytesOf(7, 32))

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO public.user_one_time_pre_keys (user_id,device_id,key_id,public_key) VALUES ($1,$2,$3,$4) ON CONFLICT (user_id, device_id, key_id) DO NOTHING`)).
		WithArgs(shared.UserID(42), deviceID.String(), userotpprekey.KeyID(11), publicKey[:]).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.AddBatch(context.Background(), []*userotpprekey.UserOTPPreKey{{
		UserID:    42,
		DeviceID:  deviceID,
		KeyID:     11,
		PublicKey: publicKey,
	}})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOTPPreKeyRepository_CountAvailableUsesUserAndDevice(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewOTPPreKeyRepository(sqlx.NewDb(db, "postgres"))
	deviceID := mustOTPRepoDeviceID(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM public.user_one_time_pre_keys WHERE device_id = $1 AND user_id = $2`)).
		WithArgs(deviceID.String(), shared.UserID(42)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))

	count, err := repo.CountAvailable(context.Background(), 42, deviceID)

	require.NoError(t, err)
	assert.Equal(t, 4, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOTPPreKeyRepository_CountAvailablePropagatesError(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewOTPPreKeyRepository(sqlx.NewDb(db, "postgres"))
	deviceID := mustOTPRepoDeviceID(t)
	repoErr := errors.New("count unavailable")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM public.user_one_time_pre_keys WHERE device_id = $1 AND user_id = $2`)).
		WithArgs(deviceID.String(), shared.UserID(42)).
		WillReturnError(repoErr)

	count, err := repo.CountAvailable(context.Background(), 42, deviceID)

	require.Error(t, err)
	assert.Zero(t, count)
	assert.ErrorIs(t, err, repoErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

func mustOTPRepoDeviceID(t *testing.T) shared.DeviceID {
	t.Helper()
	deviceID, err := shared.ParseDeviceID("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	return deviceID
}

func otpRepoBytesOf(value byte, length int) []byte {
	out := make([]byte, length)
	for i := range out {
		out[i] = value
	}
	return out
}

package e2ee

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSenderKeyRepository_AddUses64BitChainIDMirror(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewSenderKeyRepository(sqlx.NewDb(db, "postgres"))
	const senderKeyVersion = int64(1775758701055)
	const senderDeviceID = "11111111-1111-1111-1111-111111111111"
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public.member_sender_keys (chat_member_id,sender_device_id,chain_id,sender_key_version,key_fingerprint) VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at`)).
		WithArgs(int64(chatmember.ID(301)), senderDeviceID, membersenderkey.ChainID(senderKeyVersion), senderKeyVersion, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(1, now))

	sk := &membersenderkey.MemberSenderKey{
		ChatMemberID:     chatmember.ID(301),
		SenderDeviceID:   mustSenderKeyRepoDeviceID(senderDeviceID),
		SenderKeyVersion: senderKeyVersion,
	}

	err := repo.Add(context.Background(), sk)

	require.NoError(t, err)
	assert.Equal(t, membersenderkey.ID(1), sk.ID)
	assert.WithinDuration(t, now, sk.CreatedAt, time.Second)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSenderKeyRepository_UpsertLatestUsesConflictKeyOnMemberAndVersion(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewSenderKeyRepository(sqlx.NewDb(db, "postgres"))
	const senderKeyVersion = int64(1776018315645)
	const senderDeviceID = "22222222-2222-2222-2222-222222222222"
	now := time.Now()

	mock.ExpectQuery(`(?s)INSERT INTO public\.member_sender_keys \(chat_member_id,sender_device_id,chain_id,sender_key_version,key_fingerprint\) VALUES \(\$1,\$2,\$3,\$4,\$5\) ON CONFLICT \(chat_member_id, sender_device_id, sender_key_version\) DO UPDATE SET.*RETURNING id, created_at`).
		WithArgs(int64(chatmember.ID(302)), senderDeviceID, membersenderkey.ChainID(senderKeyVersion), senderKeyVersion, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(2, now))

	sk := &membersenderkey.MemberSenderKey{
		ChatMemberID:     chatmember.ID(302),
		SenderDeviceID:   mustSenderKeyRepoDeviceID(senderDeviceID),
		SenderKeyVersion: senderKeyVersion,
	}

	err := repo.UpsertLatest(context.Background(), sk)

	require.NoError(t, err)
	assert.Equal(t, membersenderkey.ID(2), sk.ID)
	assert.WithinDuration(t, now, sk.CreatedAt, time.Second)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSenderKeyRepository_FindLatestForMemberUsesPostgresPlaceholdersAndOrdering(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewSenderKeyRepository(sqlx.NewDb(db, "postgres"))
	const senderDeviceID = "11111111-1111-1111-1111-111111111111"
	now := time.Now()

	mock.ExpectQuery(`SELECT DISTINCT ON \(sender_device_id\) id, chat_member_id, sender_device_id, chain_id, sender_key_version, key_fingerprint, created_at FROM public\.member_sender_keys WHERE chat_member_id = \$1 ORDER BY sender_device_id, sender_key_version DESC, chain_id DESC`).
		WithArgs(int64(chatmember.ID(301))).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "chat_member_id", "sender_device_id", "chain_id", "sender_key_version", "key_fingerprint", "created_at",
		}).AddRow(9, int64(chatmember.ID(301)), senderDeviceID, int64(101), int64(101), nil, now))

	keys, err := repo.FindLatestForMember(context.Background(), chatmember.ID(301))

	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Equal(t, chatmember.ID(301), keys[0].ChatMemberID)
	assert.Equal(t, mustSenderKeyRepoDeviceID(senderDeviceID), keys[0].SenderDeviceID)
	assert.Equal(t, int64(101), keys[0].SenderKeyVersion)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSenderKeyRepository_FindAllByMembersUsesPostgresPlaceholdersAndOrdering(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewSenderKeyRepository(sqlx.NewDb(db, "postgres"))
	const senderDeviceID = "22222222-2222-2222-2222-222222222222"
	now := time.Now()

	mock.ExpectQuery(`SELECT DISTINCT ON \(chat_member_id, sender_device_id\) id, chat_member_id, sender_device_id, chain_id, sender_key_version, key_fingerprint, created_at FROM public\.member_sender_keys WHERE chat_member_id IN \(\$1,\$2\) ORDER BY chat_member_id, sender_device_id, sender_key_version DESC, chain_id DESC`).
		WithArgs(int64(chatmember.ID(301)), int64(chatmember.ID(302))).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "chat_member_id", "sender_device_id", "chain_id", "sender_key_version", "key_fingerprint", "created_at",
		}).AddRow(11, int64(chatmember.ID(302)), senderDeviceID, int64(202), int64(202), nil, now))

	keys, err := repo.FindAllByMembers(context.Background(), []chatmember.ID{chatmember.ID(301), chatmember.ID(302)})

	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Equal(t, chatmember.ID(302), keys[0].ChatMemberID)
	assert.Equal(t, mustSenderKeyRepoDeviceID(senderDeviceID), keys[0].SenderDeviceID)
	assert.Equal(t, int64(202), keys[0].SenderKeyVersion)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSenderKeyDistributionRepository_UpsertAvailableWrites64BitChainIDMirror(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewSenderKeyDistributionRepository(sqlx.NewDb(db, "postgres"))
	const senderKeyVersion = int64(1775758701055)
	const senderDeviceID = "33333333-3333-3333-3333-333333333333"
	now := time.Now()

	mock.ExpectQuery(`(?s)INSERT INTO public\.sender_key_distributions\s+\(sender_member_id, sender_device_id, receiver_member_id, receiver_device_id, sender_key_version, chain_id, distribution_message, status, distributed_at, consumed_at, failed_at\).*RETURNING id, distributed_at`).
		WithArgs(
			int64(chatmember.ID(401)),
			senderDeviceID,
			int64(chatmember.ID(402)),
			"00000000-0000-0000-0000-000000000000",
			senderKeyVersion,
			senderKeyVersion,
			[]byte("dist"),
			senderkeydistribution.StatusAvailable,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "distributed_at"}).AddRow(7, now))

	dist := &senderkeydistribution.SenderKeyDistribution{
		SenderMemberID:      chatmember.ID(401),
		SenderDeviceID:      mustSenderKeyRepoDeviceID(senderDeviceID),
		ReceiverMemberID:    chatmember.ID(402),
		SenderKeyVersion:    senderKeyVersion,
		ChainID:             senderKeyVersion,
		DistributionMessage: []byte("dist"),
	}

	err := repo.UpsertAvailable(context.Background(), dist)

	require.NoError(t, err)
	assert.Equal(t, senderkeydistribution.ID(7), dist.ID)
	assert.Equal(t, senderkeydistribution.StatusAvailable, dist.Status)
	assert.WithinDuration(t, now, dist.DistributedAt, time.Second)
	require.NoError(t, mock.ExpectationsWereMet())
}

func mustSenderKeyRepoDeviceID(raw string) shared.DeviceID {
	id, err := shared.ParseDeviceID(raw)
	if err != nil {
		panic(err)
	}
	return id
}

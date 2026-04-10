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
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSenderKeyRepository_AddUses64BitChainIDMirror(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewSenderKeyRepository(sqlx.NewDb(db, "postgres"))
	const senderKeyVersion = int64(1775758701055)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public.member_sender_keys (chat_member_id,chain_id,sender_key_version,key_fingerprint) VALUES ($1,$2,$3,$4) RETURNING id, created_at`)).
		WithArgs(int64(chatmember.ID(301)), membersenderkey.ChainID(senderKeyVersion), senderKeyVersion, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(1, now))

	sk := &membersenderkey.MemberSenderKey{
		ChatMemberID:     chatmember.ID(301),
		SenderKeyVersion: senderKeyVersion,
	}

	err := repo.Add(context.Background(), sk)

	require.NoError(t, err)
	assert.Equal(t, membersenderkey.ID(1), sk.ID)
	assert.WithinDuration(t, now, sk.CreatedAt, time.Second)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSenderKeyDistributionRepository_UpsertAvailableWrites64BitChainIDMirror(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewSenderKeyDistributionRepository(sqlx.NewDb(db, "postgres"))
	const senderKeyVersion = int64(1775758701055)
	now := time.Now()

	mock.ExpectQuery(`(?s)INSERT INTO public\.sender_key_distributions\s+\(sender_member_id, receiver_member_id, sender_key_version, chain_id, distribution_message, status, distributed_at, consumed_at, failed_at\).*RETURNING id, distributed_at`).
		WithArgs(
			int64(chatmember.ID(401)),
			int64(chatmember.ID(402)),
			senderKeyVersion,
			senderKeyVersion,
			[]byte("dist"),
			senderkeydistribution.StatusAvailable,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "distributed_at"}).AddRow(7, now))

	dist := &senderkeydistribution.SenderKeyDistribution{
		SenderMemberID:      chatmember.ID(401),
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

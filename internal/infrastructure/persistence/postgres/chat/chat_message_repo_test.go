package chat

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmessage"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestChatMessageRepository_FindLatestByRoomExcludingSendersAddsNotInFilter(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewChatMessageRepository(sqlx.NewDb(db, "postgres"))
	now := time.Now()

	t.Log("Given: blocked sender member ids should be excluded from latest-message lookup")
	t.Log("Input: room_id=77 excluded_sender_ids=[12,13]")
	t.Log("Action: execute FindLatestByRoomExcludingSenders")

	rows := sqlmock.NewRows(chatMessageRowColumns()).
		AddRow(31, 77, 14, "11111111-1111-1111-1111-111111111111", 1776095192099, "ciphertext", string(chatmessage.Text), nil, false, false, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, room_id, sender_id, sender_device_id, sender_key_version, content, message_type, reply_to_id, is_edited, is_deleted, created_at, updated_at FROM public.chat_records WHERE (is_deleted = $1 AND room_id = $2 AND sender_id NOT IN ($3,$4)) ORDER BY id DESC LIMIT 1`)).
		WithArgs(false, chatroom.ID(77), int64(12), int64(13)).
		WillReturnRows(rows)

	msg, err := repo.FindLatestByRoomExcludingSenders(
		context.Background(),
		chatroom.ID(77),
		[]chatmember.ID{12, 13},
	)

	t.Logf("Output: message_id=%d sender_id=%d", msg.ID, msg.SenderID)
	t.Log("Mutation: none")
	require.NoError(t, err)
	require.Equal(t, chatmember.ID(14), msg.SenderID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func chatMessageRowColumns() []string {
	return []string{
		"id",
		"room_id",
		"sender_id",
		"sender_device_id",
		"sender_key_version",
		"content",
		"message_type",
		"reply_to_id",
		"is_edited",
		"is_deleted",
		"created_at",
		"updated_at",
	}
}

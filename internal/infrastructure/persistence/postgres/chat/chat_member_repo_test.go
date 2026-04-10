package chat

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestChatMemberRepository_FindActiveMembersFiltersDeletedRows(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewChatMemberRepository(sqlx.NewDb(db, "postgres"))
	now := time.Now()

	t.Log("Given: chat member list queries should hide soft-deleted members")
	t.Log("Input: room_id=77 participant_id=88")
	t.Log("Action: execute FindByRoom and FindByParticipant")

	roomRows := sqlmock.NewRows(chatMemberRowColumns()).
		AddRow(11, 77, 88, string(chatmember.Member), false, false, nil, now, now, nil)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, room_id, participant_id, role, is_muted, is_deleted, last_read_at, joined_at, updated_at, deleted_at FROM public.chat_members WHERE is_deleted = $1 AND room_id = $2`)).
		WithArgs(false, chatroom.ID(77)).
		WillReturnRows(roomRows)

	participantRows := sqlmock.NewRows(chatMemberRowColumns()).
		AddRow(12, 78, 88, string(chatmember.Owner), false, false, nil, now, now, nil)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, room_id, participant_id, role, is_muted, is_deleted, last_read_at, joined_at, updated_at, deleted_at FROM public.chat_members WHERE is_deleted = $1 AND participant_id = $2`)).
		WithArgs(false, participant.ID(88)).
		WillReturnRows(participantRows)

	byRoom, err := repo.FindByRoom(context.Background(), chatroom.ID(77))
	require.NoError(t, err)
	byParticipant, err := repo.FindByParticipant(context.Background(), participant.ID(88))
	require.NoError(t, err)

	t.Logf("Output: by_room=%d by_participant=%d", len(byRoom), len(byParticipant))
	t.Log("Mutation: read-only SELECT queries include is_deleted=false")
	require.Len(t, byRoom, 1)
	require.Len(t, byParticipant, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func chatMemberRowColumns() []string {
	return []string{
		"id",
		"room_id",
		"participant_id",
		"role",
		"is_muted",
		"is_deleted",
		"last_read_at",
		"joined_at",
		"updated_at",
		"deleted_at",
	}
}

package friendship

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	domainfriendship "github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/testutil"
	"github.com/jmoiron/sqlx"
)

func TestFriendshipRepo_CreateUsesPendingInsertShape(t *testing.T) {
	start := time.Now()
	db, mock := testutil.SetupDB(t)
	repo := NewFriendshipRepository(sqlx.NewDb(db, "postgres"))

	t.Log("Given: a new friendship row should be inserted as pending")
	t.Log("Input: user_id=501 friend_id=601 status=pending")
	t.Log("Action: execute FriendshipRepository.Create")

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO public.user_friendships (user_id,friend_id,status) VALUES ($1,$2,$3)`)).
		WithArgs(shared.UserID(501), shared.UserID(601), string(domainfriendship.StatusPending)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), 501, 601)

	t.Logf("Output: err=%v", err)
	t.Log("Mutation: one pending INSERT expected")
	t.Logf("Duration: %s", time.Since(start))

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFriendshipRepo_FindPendingQueriesMapRowsAndNotFound(t *testing.T) {
	start := time.Now()
	db, mock := testutil.SetupDB(t)
	repo := NewFriendshipRepository(sqlx.NewDb(db, "postgres"))

	t.Log("Given: pending sent and inbound friendship queries should filter by status")
	t.Log("Input: user_id=501")
	t.Log("Action: execute FindPendingByUserID and FindPendingByFriendID")

	rows := sqlmock.NewRows([]string{"id", "user_id", "friend_id", "status", "created_at", "updated_at"}).
		AddRow(11, 501, 601, "pending", time.Now(), time.Now())
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, friend_id, status, created_at, updated_at FROM public.user_friendships WHERE status = $1 AND user_id = $2`)).
		WithArgs(string(domainfriendship.StatusPending), shared.UserID(501)).
		WillReturnRows(rows)

	inboundRows := sqlmock.NewRows([]string{"id", "user_id", "friend_id", "status", "created_at", "updated_at"}).
		AddRow(12, 602, 501, "pending", time.Now(), time.Now())
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, friend_id, status, created_at, updated_at FROM public.user_friendships WHERE friend_id = $1 AND status = $2`)).
		WithArgs(shared.UserID(501), string(domainfriendship.StatusPending)).
		WillReturnRows(inboundRows)

	sent, err := repo.FindPendingByUserID(context.Background(), 501)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	inbound, err := repo.FindPendingByFriendID(context.Background(), 501)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Logf("Output: sent=%d inbound=%d", len(sent), len(inbound))
	t.Log("Mutation: read-only SELECT queries")
	t.Logf("Duration: %s", time.Since(start))

	if len(sent) != 1 || len(inbound) != 1 {
		t.Fatalf("expected one sent and one inbound row, got sent=%d inbound=%d", len(sent), len(inbound))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFriendshipRepo_FindBetweenUsersEmptyMapsNotFound(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewFriendshipRepository(sqlx.NewDb(db, "postgres"))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, friend_id, status, created_at, updated_at FROM public.user_friendships WHERE ((user_id = $1 AND friend_id = $2) OR (user_id = $3 AND friend_id = $4))`)).
		WithArgs(shared.UserID(501), shared.UserID(601), shared.UserID(601), shared.UserID(501)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "friend_id", "status", "created_at", "updated_at"}))

	_, err := repo.FindBetweenUsers(context.Background(), 501, 601)
	if !errors.Is(err, domainfriendship.ErrFriendshipNotFound) {
		t.Fatalf("expected ErrFriendshipNotFound, got %v", err)
	}
}

func TestFriendshipRepo_UpdateStatusAndDeleteUseExpectedSQL(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewFriendshipRepository(sqlx.NewDb(db, "postgres"))

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE public.user_friendships SET status = $1, updated_at = now() WHERE id = $2`)).
		WithArgs(string(domainfriendship.StatusAccepted), int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM public.user_friendships WHERE id = $1`)).
		WithArgs(int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpdateStatus(context.Background(), 10, domainfriendship.StatusAccepted); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := repo.Delete(context.Background(), 10); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

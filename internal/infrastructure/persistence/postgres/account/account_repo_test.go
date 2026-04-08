package account

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	domainaccount "github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/testutil"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

func TestAccountRepo_CreateSuccessHasStructuredLog(t *testing.T) {
	start := time.Now()
	db, mock := testutil.SetupDB(t)
	repo := NewAccountRepo(sqlx.NewDb(db, "postgres"))
	acc := domainaccount.NewAccount(uuid.Nil, shared.EmailAddress("new@example.com"), "new_account", "argon2-hash", 1)

	t.Log("Given: account insert returns a generated id")
	t.Logf("Input: email=%s account=%s status=%s user_limit=%d password_hash_present=%t", acc.Email, acc.AccountName, acc.Status, acc.UserLimit, acc.Password != "")
	t.Log("Action: execute AccountRepo.Create")

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public.accounts (public_id,email,account,password,status,user_limit) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`)).
		WithArgs(acc.PublicID, acc.Email, acc.AccountName, acc.Password, acc.Status, acc.UserLimit).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(101))

	id, err := repo.Create(context.Background(), acc)

	t.Logf("Output: id=%d err=%v", id, err)
	t.Log("Mutation: one INSERT statement expected")
	t.Logf("Duration: %s", time.Since(start))

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != 101 {
		t.Fatalf("expected id 101, got %d", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAccountRepo_CreateDuplicateEmailHasStructuredLog(t *testing.T) {
	start := time.Now()
	db, mock := testutil.SetupDB(t)
	repo := NewAccountRepo(sqlx.NewDb(db, "postgres"))
	acc := domainaccount.NewAccount(uuid.Nil, shared.EmailAddress("taken@example.com"), "new_account", "argon2-hash", 1)

	t.Log("Given: account insert violates the email unique constraint")
	t.Logf("Input: email=%s account=%s password_hash_present=%t", acc.Email, acc.AccountName, acc.Password != "")
	t.Log("Action: execute AccountRepo.Create duplicate-email path")

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public.accounts (public_id,email,account,password,status,user_limit) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`)).
		WithArgs(acc.PublicID, acc.Email, acc.AccountName, acc.Password, acc.Status, acc.UserLimit).
		WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "accounts_email_key"})

	id, err := repo.Create(context.Background(), acc)

	t.Logf("Output: id=%d err=%v", id, err)
	t.Log("Mutation: one INSERT statement expected and no rows inserted")
	t.Logf("Duration: %s", time.Since(start))

	if !errors.Is(err, domainaccount.ErrEmailExist) {
		t.Fatalf("expected ErrEmailExist, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAccountRepo_CreateDuplicateAccountHasStructuredLog(t *testing.T) {
	start := time.Now()
	db, mock := testutil.SetupDB(t)
	repo := NewAccountRepo(sqlx.NewDb(db, "postgres"))
	acc := domainaccount.NewAccount(uuid.Nil, shared.EmailAddress("new@example.com"), "taken_account", "argon2-hash", 1)

	t.Log("Given: account insert violates the account unique constraint")
	t.Logf("Input: email=%s account=%s password_hash_present=%t", acc.Email, acc.AccountName, acc.Password != "")
	t.Log("Action: execute AccountRepo.Create duplicate-account path")

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public.accounts (public_id,email,account,password,status,user_limit) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`)).
		WithArgs(acc.PublicID, acc.Email, acc.AccountName, acc.Password, acc.Status, acc.UserLimit).
		WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "accounts_account_key"})

	id, err := repo.Create(context.Background(), acc)

	t.Logf("Output: id=%d err=%v", id, err)
	t.Log("Mutation: one INSERT statement expected and no rows inserted")
	t.Logf("Duration: %s", time.Since(start))

	if !errors.Is(err, domainaccount.ErrAccountExist) {
		t.Fatalf("expected ErrAccountExist, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAccountRepo_FindByEmailNotFoundMapsDomainErrorHasStructuredLog(t *testing.T) {
	start := time.Now()
	db, mock := testutil.SetupDB(t)
	repo := NewAccountRepo(sqlx.NewDb(db, "postgres"))
	email := shared.EmailAddress("missing@example.com")

	t.Log("Given: no account row exists for the requested email")
	t.Logf("Input: email=%s", email)
	t.Log("Action: execute AccountRepo.FindByEmail not-found path")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, public_id, email, account, password, status, user_limit, created_at, updated_at FROM public.accounts WHERE email = $1 LIMIT 1`)).
		WithArgs(email).
		WillReturnError(sql.ErrNoRows)

	got, err := repo.FindByEmail(context.Background(), email)

	t.Logf("Output: account_nil=%t err=%v", got == nil, err)
	t.Log("Mutation: read-only SELECT account row")
	t.Logf("Duration: %s", time.Since(start))

	if !errors.Is(err, domainaccount.ErrAccountNotFound) {
		t.Fatalf("expected ErrAccountNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAccountRepo_FindByAccountNameNotFoundMapsDomainErrorHasStructuredLog(t *testing.T) {
	start := time.Now()
	db, mock := testutil.SetupDB(t)
	repo := NewAccountRepo(sqlx.NewDb(db, "postgres"))
	accountName := "missing_account"

	t.Log("Given: no account row exists for the requested account name")
	t.Logf("Input: account=%s", accountName)
	t.Log("Action: execute AccountRepo.FindByAccountName not-found path")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, public_id, email, account, password, status, user_limit, created_at, updated_at FROM public.accounts WHERE account = $1 LIMIT 1`)).
		WithArgs(accountName).
		WillReturnError(sql.ErrNoRows)

	got, err := repo.FindByAccountName(context.Background(), accountName)

	t.Logf("Output: account_nil=%t err=%v", got == nil, err)
	t.Log("Mutation: read-only SELECT account row")
	t.Logf("Duration: %s", time.Since(start))

	if !errors.Is(err, domainaccount.ErrAccountNotFound) {
		t.Fatalf("expected ErrAccountNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

package user

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/testutil"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logger.InitTestEnv()
	m.Run()
}

func TestUserRepository_Create_Success(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewUserRepository(sqlx.NewDb(db, "postgres"))

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO goat.public.users (account_id,name) VALUES ($1,$2) RETURNING id`)).
		WithArgs(int64(1), "hiro").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))

	u := user.NewUser(1, "hiro")
	id, err := repo.Create(context.Background(), u)
	assert.NoError(t, err)
	assert.Equal(t, user.ID(10), id)
	assert.Equal(t, user.ID(10), u.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Create_Duplicate(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewUserRepository(sqlx.NewDb(db, "postgres"))

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO goat.public.users (account_id,name) VALUES ($1,$2) RETURNING id`)).
		WithArgs(int64(1), "hiro").
		WillReturnError(errors.New(`duplicate key value violates unique constraint "users_account_id_key"`))

	u := user.NewUser(1, "hiro")
	_, err := repo.Create(context.Background(), u)
	assert.ErrorIs(t, err, user.ErrUserAlreadyExists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_FindByID(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewUserRepository(sqlx.NewDb(db, "postgres"))

	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, account_id, name, avatar, created_at, updated_at FROM goat.public.users WHERE id = $1 LIMIT 1`)).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "account_id", "name", "avatar", "created_at", "updated_at"}).
			AddRow(10, 1, "hiro", nil, now, now))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT roles.code FROM goat.public.users_roles JOIN roles ON roles.id = users_roles.role_id WHERE user_id = $1`)).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"code"}))

	got, err := repo.FindByID(context.Background(), 10)
	assert.NoError(t, err)
	assert.Equal(t, user.ID(10), got.ID)
	assert.Equal(t, "hiro", got.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

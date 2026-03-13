package user

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/HiroLiang/goat-server/internal/domain/role"
	"github.com/HiroLiang/goat-server/internal/domain/user"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestUserRepository_CreateWithRole_Success(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewUserRepository(sqlx.NewDb(db, "postgres"))

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public.users (name,email,password,user_status,user_ip) VALUES ($1,$2,$3,$4,$5) RETURNING id`)).
		WithArgs("hiro", "hiro@gmail.com", "hash", "applying", "127.0.0.1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM goat.public.roles WHERE type = $1 LIMIT 1`)).
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO goat.public.users_roles (user_id,role_id) VALUES ($1,$2)`)).
		WithArgs(int64(10), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	u := user.NewUser("hiro", user.Email("hiro@gmail.com"), "hash", "127.0.0.1")
	err := repo.CreateWithRole(context.Background(), u, string(role.User))
	assert.NoError(t, err)
	assert.Equal(t, user.ID(10), u.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_CreateWithRole_DefaultRoleMissing(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewUserRepository(sqlx.NewDb(db, "postgres"))

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public.users (name,email,password,user_status,user_ip) VALUES ($1,$2,$3,$4,$5) RETURNING id`)).
		WithArgs("hiro", "hiro@gmail.com", "hash", "applying", "127.0.0.1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM goat.public.roles WHERE type = $1 LIMIT 1`)).
		WithArgs("user").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	u := user.NewUser("hiro", user.Email("hiro@gmail.com"), "hash", "127.0.0.1")
	err := repo.CreateWithRole(context.Background(), u, string(role.User))
	assert.ErrorIs(t, err, user.ErrDefaultRoleNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

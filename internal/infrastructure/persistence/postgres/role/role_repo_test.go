package role

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestRoleRepository_FindByCode(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewRoleRepository(sqlx.NewDb(db, "postgres"))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, code, name, description, created_by, created_at, updated_at FROM goat.public.roles WHERE code = $1 LIMIT 1`)).
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "name", "description", "created_by", "created_at", "updated_at"}).
			AddRow(1, "user", "User", nil, int64(1), time.Now(), time.Now()))

	got, err := repo.FindByCode(context.Background(), role.User)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Equal(t, role.User, got.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_Create_Duplicate(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewRoleRepository(sqlx.NewDb(db, "postgres"))

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO goat.public.roles (code,name,description,created_by) VALUES ($1,$2,$3,$4)`)).
		WithArgs("user", "User", (*string)(nil), (*int64)(nil)).
		WillReturnError(errors.New(`duplicate key value violates unique constraint "roles_code_key"`))

	err := repo.Create(context.Background(), &role.Role{Code: role.User, Name: "User"})
	assert.ErrorIs(t, err, role.ErrAlreadyExists)
}

func TestRoleRepository_FindAll(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewRoleRepository(sqlx.NewDb(db, "postgres"))

	mock.ExpectQuery(`SELECT .* FROM goat.public.roles`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "name", "description", "created_by", "created_at", "updated_at"}).
			AddRow(1, "user", "User", nil, int64(1), time.Now(), time.Now()).
			AddRow(2, "admin", "Administrator", nil, int64(1), time.Now(), time.Now()))

	roles, err := repo.FindAll(context.Background())
	assert.NoError(t, err)
	assert.Len(t, roles, 2)
	assert.Equal(t, role.Admin, roles[1].Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_Update_NotFound(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewRoleRepository(sqlx.NewDb(db, "postgres"))

	mock.ExpectQuery(regexp.QuoteMeta(
		`UPDATE goat.public.roles SET code = $1, name = $2, description = $3, created_by = $4, updated_at = now() WHERE id = $5 RETURNING id`,
	)).
		WithArgs("user", "User", (*string)(nil), (*int64)(nil), int64(999)).
		WillReturnError(sql.ErrNoRows)

	err := repo.Update(context.Background(), &role.Role{ID: 999, Code: role.User, Name: "User"})
	assert.ErrorIs(t, err, role.ErrNotFound)
}

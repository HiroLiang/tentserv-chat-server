package role

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/HiroLiang/goat-server/internal/domain/role"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestRoleRepository_FindByType(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewRoleRepository(sqlx.NewDb(db, "postgres"))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, type, creator, created_at, updated_at FROM goat.public.roles WHERE type = $1 LIMIT 1`)).
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "creator", "created_at", "updated_at"}).
			AddRow(1, "user", 1, time.Now(), time.Now()))

	got, err := repo.FindByType(context.Background(), role.User)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Equal(t, role.User, got.Type)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_Create_Duplicate(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewRoleRepository(sqlx.NewDb(db, "postgres"))

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO goat.public.roles (type,creator) VALUES ($1,$2)`)).
		WithArgs("user", int64(1)).
		WillReturnError(errors.New(`duplicate key value violates unique constraint "roles_type_key"`))

	err := repo.Create(context.Background(), &role.Role{Type: role.User, Creator: 1})
	assert.ErrorIs(t, err, role.ErrAlreadyExists)
}

func TestRoleRepository_FindAll(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewRoleRepository(sqlx.NewDb(db, "postgres"))

	mock.ExpectQuery(`SELECT .* FROM goat.public.roles`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "creator", "created_at", "updated_at"}).
			AddRow(1, "user", 1, time.Now(), time.Now()).
			AddRow(2, "admin", 1, time.Now(), time.Now()))

	roles, err := repo.FindAll(context.Background())
	assert.NoError(t, err)
	assert.Len(t, roles, 2)
	assert.Equal(t, role.Admin, roles[1].Type)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_Update_NotFound(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := NewRoleRepository(sqlx.NewDb(db, "postgres"))

	mock.ExpectQuery(regexp.QuoteMeta(
		`UPDATE goat.public.roles SET type = $1, creator = $2, updated_at = now() WHERE id = $3 RETURNING id`,
	)).
		WithArgs("user", int64(1), int64(999)).
		WillReturnError(sql.ErrNoRows)

	err := repo.Update(context.Background(), &role.Role{ID: 999, Type: role.User, Creator: 1})
	assert.ErrorIs(t, err, role.ErrNotFound)
}

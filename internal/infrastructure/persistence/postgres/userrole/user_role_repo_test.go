package userrole

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/HiroLiang/goat-server/internal/domain/role"
	"github.com/HiroLiang/goat-server/internal/domain/user"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

// TestUserRoleRepository_Exists Test is the Exist query correct
func TestUserRoleRepository_Exists(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := UserRoleRepository{db: sqlx.NewDb(db, "postgres")}

	mock.ExpectQuery(`SELECT 1 .*users_roles.*roles.*`).
		WithArgs(user.ID(1), "admin").
		WillReturnRows(
			sqlmock.NewRows([]string{"1"}).AddRow(1),
		)

	ok := repo.Exists(context.Background(), user.ID(1), role.Admin)
	assert.True(t, ok)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRoleRepository_FindRolesByUser(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := UserRoleRepository{db: sqlx.NewDb(db, "postgres")}

	mock.ExpectQuery(`SELECT r.id, r.code, r.name, r.description, r.created_by, r.created_at, r.updated_at FROM .*roles.* JOIN .*users_roles.*`).
		WithArgs(user.ID(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "name", "description", "created_by", "created_at", "updated_at"}).
			AddRow(1, "user", "User", nil, int64(1), time.Now(), time.Now()).
			AddRow(2, "admin", "Administrator", nil, int64(1), time.Now(), time.Now()))

	roles, err := repo.FindRolesByUser(context.Background(), user.ID(1))
	assert.NoError(t, err)
	assert.Len(t, roles, 2)
	if len(roles) < 2 {
		return
	}
	assert.Equal(t, role.User, roles[0].Code)
	assert.Equal(t, role.Admin, roles[1].Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUserRoleRepository_Assign Test is the Assign query correct
func TestUserRoleRepository_Assign(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := UserRoleRepository{db: sqlx.NewDb(db, "postgres")}

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO goat.public.users_roles (user_id,role_id)
		SELECT $1, id FROM goat.public.roles WHERE code = $2
		ON CONFLICT DO NOTHING RETURNING user_id
	`)).
		WithArgs(user.ID(1), "admin").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(1))

	err := repo.Assign(context.Background(), user.ID(1), role.Admin)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRoleRepository_Revoke(t *testing.T) {
	db, mock := testutil.SetupDB(t)
	repo := UserRoleRepository{db: sqlx.NewDb(db, "postgres")}

	//noinspection SqlWithoutWhere
	mock.ExpectExec(
		`DELETE FROM.*users_roles.* WHERE .* USING .*roles.*`,
	).
		WithArgs("admin", user.ID(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Revoke(context.Background(), user.ID(1), role.Admin)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

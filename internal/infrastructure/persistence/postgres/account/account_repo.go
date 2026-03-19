package account

import (
	"context"
	"errors"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres/user"
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type AccountRepo struct {
	postgres.BaseRepo
}

func NewAccountRepo(db *sqlx.DB) *AccountRepo {
	return &AccountRepo{
		BaseRepo: postgres.NewBaseRepo(db),
	}
}

func (r *AccountRepo) FindByID(ctx context.Context, id shared.AccountID) (*account.Account, error) {
	return r.findAccount(ctx, squirrel.Eq{"id": id})
}

func (r *AccountRepo) FindByAccountName(ctx context.Context, accountName string) (*account.Account, error) {
	return r.findAccount(ctx, squirrel.Eq{"account": accountName})
}

func (r *AccountRepo) FindByEmail(ctx context.Context, email shared.EmailAddress) (*account.Account, error) {
	return r.findAccount(ctx, squirrel.Eq{"email": email})

}

func (r *AccountRepo) Create(ctx context.Context, accountData *account.Account) (shared.AccountID, error) {
	query, args, err := Table.Insert().
		Columns("public_id", "email", "account", "password", "status", "user_limit").
		Values(
			accountData.PublicID, accountData.Email, accountData.AccountName, accountData.Password, accountData.Status, accountData.UserLimit).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return -1, err
	}

	var id int64
	err = r.GetDB(ctx).QueryRowxContext(ctx, query, args...).Scan(&id)
	if err != nil {

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {

			if pgErr.Code == "23505" {

				switch pgErr.ConstraintName {
				case "accounts_email_key":
					return -1, account.ErrEmailExist

				case "accounts_account_key":
					return -1, account.ErrAccountExist
				}
			}
		}

		return -1, err
	}

	return shared.AccountID(id), nil
}

func (r *AccountRepo) Update(ctx context.Context, account *account.Account) error {
	query, args, err := Table.Update().
		Set("email", account.Email).
		Set("account", account.AccountName).
		Set("password", account.Password).
		Set("status", account.Status).
		Set("user_limit", account.UserLimit).
		Set("updated_at", squirrel.Expr("NOW()")).
		Where(squirrel.Eq{"id": account.ID}).
		ToSql()
	if err != nil {
		return err
	}

	err = postgres.Exec(ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return err
	}

	return nil
}

func (r *AccountRepo) RegisterDevice(ctx context.Context, accountDevice *account.AccountDevice) error {
	query, args, err := JoinTable.Insert().
		Columns("account_id", "device_id", "last_ip", "last_seen_at").
		Values(
			accountDevice.AccountID,
			accountDevice.DeviceID,
			accountDevice.LastIP.String(),
			squirrel.Expr("NOW()"),
		).
		Suffix(`
            ON CONFLICT (account_id, device_id)
            DO UPDATE SET
                last_ip = EXCLUDED.last_ip,
                last_seen_at = EXCLUDED.last_seen_at
        `).
		ToSql()
	if err != nil {
		return err
	}

	return postgres.Exec(ctx, r.GetDB(ctx), query, args...)
}

func (r *AccountRepo) ReplaceDevices(
	ctx context.Context,
	accountID shared.AccountID,
	devices []account.AccountDevice,
) error {
	db := r.GetDB(ctx)

	// delete old
	query, args, err := JoinTable.Delete().
		Where(squirrel.Eq{"account_id": accountID}).
		ToSql()
	if err != nil {
		return err
	}

	if err := postgres.Exec(ctx, db, query, args...); err != nil {
		return err
	}

	if len(devices) == 0 {
		return nil
	}

	// Insert new
	builder := JoinTable.Insert().
		Columns("account_id", "device_id", "last_ip", "last_seen_at")

	for _, d := range devices {
		builder = builder.Values(
			accountID,
			d.DeviceID,
			d.LastIP.String(),
			d.LastSeenAt,
		)
	}

	query, args, err = builder.ToSql()
	if err != nil {
		return err
	}

	return postgres.Exec(ctx, db, query, args...)
}

func (r *AccountRepo) findAccount(
	ctx context.Context,
	cond squirrel.Eq,
) (*account.Account, error) {
	query, args, err := Table.
		Select(Table.Columns...).
		Where(cond).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, err
	}

	record, err := postgres.ScanOne[AccountRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return nil, err
	}

	accountData := ToAccount(record)

	userIDs, err := r.getUserIDs(ctx, accountData.ID)
	if err != nil {
		return nil, err
	}

	devices, err := r.getAccountDevices(ctx, accountData.ID)
	if err != nil {
		return nil, err
	}

	accountData.UserIDs = userIDs
	accountData.Devices = devices
	return accountData, nil
}

func (r *AccountRepo) getUserIDs(ctx context.Context, accountID shared.AccountID) ([]shared.UserID, error) {
	query, args, err := user.Table.
		Select("id").
		Where(squirrel.Eq{"account_id": accountID}).
		ToSql()
	if err != nil {
		return nil, err
	}

	records, err := postgres.ScanAll[int64](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return nil, err
	}

	userIDs := make([]shared.UserID, 0, len(records))
	for _, record := range records {
		userIDs = append(userIDs, shared.UserID(record))
	}

	return userIDs, nil
}

func (r *AccountRepo) getAccountDevices(ctx context.Context, accountID shared.AccountID) ([]account.AccountDevice, error) {
	query, args, err := JoinTable.Select(JoinTable.Columns...).
		Where(squirrel.Eq{"account_id": accountID}).
		ToSql()
	if err != nil {
		return nil, err
	}

	list, err := postgres.ScanAll[AccountDeviceRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return nil, err
	}

	devices := make([]account.AccountDevice, 0, len(list))
	for _, record := range list {
		devices = append(devices, *ToAccountDevice(&record))
	}

	return devices, nil
}

var _ account.Repository = (*AccountRepo)(nil)

var Table = postgres.Table{
	Name: "goat.public.accounts",
	Columns: []string{
		"id",
		"public_id",
		"email",
		"account",
		"password",
		"status",
		"user_limit",
		"created_at",
		"updated_at",
	},
}

var JoinTable = postgres.Table{
	Name: "goat.public.accounts_devices",
	Columns: []string{
		"account_id",
		"device_id",
		"last_ip",
		"last_seen_at",
	},
}

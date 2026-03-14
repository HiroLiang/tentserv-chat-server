package account

import (
	"time"

	"github.com/gofrs/uuid"
)

type AccountRecord struct {
	ID        int64     `db:"id"`
	PublicID  uuid.UUID `db:"public_id"`
	Email     string    `db:"email"`
	Account   string    `db:"account"`
	Password  string    `db:"password"`
	Status    string    `db:"status"`
	UserLimit int64     `db:"user_limit"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type AccountDeviceRecord struct {
	AccountID  int64     `db:"account_id"`
	DeviceID   uuid.UUID `db:"device_id"`
	LastIP     string    `db:"last_ip"`
	LastSeenAt time.Time `db:"last_seen_at"`
}

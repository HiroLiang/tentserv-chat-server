package device

import "time"

type DeviceRecord struct {
	ID        string    `db:"id"`
	Platform  string    `db:"platform"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

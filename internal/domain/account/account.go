package account

import (
	"time"

	"github.com/HiroLiang/goat-server/internal/domain/shared"
	"github.com/gofrs/uuid"
)

type Account struct {
	ID          shared.AccountID
	PublicID    uuid.UUID
	Email       shared.EmailAddress
	AccountName string
	Password    string
	Status      Status
	UserLimit   int
	UserIDs     []shared.UserID
	DeviceIDs   []shared.DeviceID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewAccount(
	publicID uuid.UUID,
	email shared.EmailAddress,
	account string,
	password string,
) *Account {
	return &Account{
		PublicID:    publicID,
		Email:       email,
		AccountName: account,
		Password:    password,
		Status:      Applying,
	}
}

func (a *Account) SetStatus(status Status) {
	a.Status = status
}

func (a *Account) HasUser(userID shared.UserID) bool {
	for _, id := range a.UserIDs {
		if id == userID {
			return true
		}
	}
	return false
}

func (a *Account) AddUser(userID shared.UserID) {
	a.UserIDs = append(a.UserIDs, userID)
}

func (a *Account) HasDevice(deviceID shared.DeviceID) bool {
	for _, id := range a.DeviceIDs {
		if id == deviceID {
			return true
		}
	}
	return false
}

func (a *Account) AddDevice(deviceID shared.DeviceID) {
	a.DeviceIDs = append(a.DeviceIDs, deviceID)
}

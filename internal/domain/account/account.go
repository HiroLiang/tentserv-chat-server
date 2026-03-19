package account

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/gofrs/uuid"
)

type Account struct {
	ID          shared.AccountID
	PublicID    uuid.UUID
	Email       shared.EmailAddress
	AccountName string
	Password    string
	Status      Status
	UserLimit   int64
	UserIDs     []shared.UserID
	Devices     []AccountDevice
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewAccount(
	publicID uuid.UUID,
	email shared.EmailAddress,
	account string,
	password string,
	userLimit int64,
) *Account {
	return &Account{
		PublicID:    publicID,
		Email:       email,
		AccountName: account,
		Password:    password,
		Status:      Applying,
		UserLimit:   userLimit,
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
	for _, device := range a.Devices {
		if device.DeviceID == deviceID {
			return true
		}
	}
	return false
}

func (a *Account) AddDevice(device AccountDevice) {
	a.Devices = append(a.Devices, device)
}

func (a *Account) GetDevice(deviceID shared.DeviceID) *AccountDevice {
	for _, device := range a.Devices {
		if device.DeviceID == deviceID {
			return &device
		}
	}
	return nil
}

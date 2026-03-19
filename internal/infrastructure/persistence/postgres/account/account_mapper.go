package account

import (
	"net"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

func ToAccount(record *AccountRecord) *account.Account {
	email, _ := shared.ParseEmail(record.Email)

	return &account.Account{
		ID:          shared.AccountID(record.ID),
		PublicID:    record.PublicID,
		Email:       email,
		AccountName: record.Account,
		Password:    record.Password,
		Status:      account.Status(record.Status),
		UserLimit:   record.UserLimit,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

func ToAccountDevice(record *AccountDeviceRecord) *account.AccountDevice {
	if record == nil {
		return nil
	}

	return &account.AccountDevice{
		AccountID:  shared.AccountID(record.AccountID),
		DeviceID:   shared.DeviceID(record.DeviceID),
		LastIP:     net.ParseIP(record.LastIP),
		LastSeenAt: record.LastSeenAt,
	}
}

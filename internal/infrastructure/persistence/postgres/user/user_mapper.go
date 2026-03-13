package user

import "github.com/HiroLiang/goat-server/internal/domain/user"

func toDomain(record *UserRecord) (*user.User, error) {
	avatarName := ""
	if record.AvatarName != nil {
		avatarName = *record.AvatarName
	}
	return &user.User{
		ID:        record.ID,
		AccountID: record.AccountID,
		Name:      record.Name,
		Avatar:    avatarName,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}, nil
}

func toRecord(u *user.User) *UserRecord {
	var avatarURL *string
	if u.Avatar != "" {
		avatarURL = &u.Avatar
	}
	return &UserRecord{
		ID:         u.ID,
		AccountID:  u.AccountID,
		Name:       u.Name,
		AvatarName: avatarURL,
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}
}

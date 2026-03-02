package user

import "github.com/HiroLiang/goat-server/internal/domain/user"

func toDomain(record *UserRecord) (*user.User, error) {
	avatarName := ""
	if record.AvatarName != nil {
		avatarName = *record.AvatarName
	}
	return &user.User{
		ID:         record.ID,
		Name:       record.Name,
		Email:      record.Email,
		Password:   record.Password,
		Status:     record.UserStatus,
		LastIP:     record.UserIP,
		AvatarName: avatarName,
		CreatedAt:  record.CreatedAt,
		UpdatedAt:  record.UpdatedAt,
	}, nil
}

func toRecord(u *user.User) *UserRecord {
	var avatarURL *string
	if u.AvatarName != "" {
		avatarURL = &u.AvatarName
	}
	return &UserRecord{
		ID:         u.ID,
		Name:       u.Name,
		Email:      u.Email,
		Password:   u.Password,
		UserStatus: u.Status,
		UserIP:     u.LastIP,
		AvatarName: avatarURL,
	}
}

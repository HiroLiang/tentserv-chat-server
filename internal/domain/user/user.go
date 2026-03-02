package user

import "time"

type User struct {
	ID         ID
	Name       string
	Email      Email
	Password   string
	Status     Status
	LastIP     string
	AvatarName string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewUser(name string, email Email, hash string, ip string) *User {
	return &User{
		Name:     name,
		Email:    email,
		Password: hash,
		Status:   Applying,
		LastIP:   ip,
	}
}

func (u *User) IsValid() bool {
	return u.ID != 0 && u.IsActive()
}

func (u *User) IsActive() bool {
	return u.Status == Active
}

package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID               uuid.UUID
	Name             string
	Email            string
	AvatarURL        string
	Bio              string
	IsVerified       bool
	IsAvailableToday bool
	LastActiveAt     time.Time
	CreatedAt        time.Time
}

func NewUser(name, email string) *User {
	return &User{
		Name:  name,
		Email: email,
	}
}

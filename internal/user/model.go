package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID               uuid.UUID
	Name             string
	AvatarURL        string
	Bio              string
	IsVerified       bool
	IsAvailableToday bool
	LastActiveAt     time.Time
	CreatedAt        time.Time
}

func NewUser(name, bio string) *User {
	return &User{
		Name: name,
		Bio:  bio,
	}
}

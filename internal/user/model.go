package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	AvatarURL        string    `json:"avatar_url"`
	Bio              string    `json:"bio"`
	IsVerified       bool      `json:"is_verified"`
	IsAvailableToday bool      `json:"is_available_today"`
	LastActiveAt     time.Time `json:"last_active_at"`
	CreatedAt        time.Time `json:"created_at"`
}

type UserSummary struct {
	ID        uuid.UUID
	Name      string
	AvatarURL string
}

func NewUser(name, email string) *User {
	return &User{
		Name:  name,
		Email: email,
	}
}

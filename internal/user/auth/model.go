package auth

import (
	"time"

	"github.com/google/uuid"
)

type UserAuth struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	GoogleSub      string
	PhoneEncrypted *string
	PhoneHMAC      *string
	CreatedAt      time.Time
}

func NewUserAuth(userID uuid.UUID, googleSub string, phoneEncrypted, phoneHMAC *string) *UserAuth {
	return &UserAuth{
		UserID:         userID,
		GoogleSub:      googleSub,
		PhoneEncrypted: phoneEncrypted,
		PhoneHMAC:      phoneHMAC,
	}
}

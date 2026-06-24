package reputation

import (
	"github.com/google/uuid"
)

type UserReputation struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RatingAsEmployer float32
	RatingAsEmployee float32
	RatingCount      int
}

func NewUserReputation(userID uuid.UUID) *UserReputation {
	return &UserReputation{
		UserID: userID,
	}
}

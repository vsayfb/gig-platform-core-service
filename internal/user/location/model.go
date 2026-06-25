package location

import (
	"time"

	"github.com/google/uuid"
)

type UserLocation struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	UpdatedAt time.Time `json:"updated_at"`
	IsFlagged bool      `json:"is_flagged"`
}

func NewUserLocation(userID uuid.UUID, lat, lng float64) *UserLocation {
	return &UserLocation{
		UserID: userID,
		Lat:    lat,
		Lng:    lng,
	}
}

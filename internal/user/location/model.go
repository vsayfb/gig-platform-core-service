package location

import (
	"time"

	"github.com/google/uuid"
)

type UserLocation struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Lat       float64
	Lng       float64
	UpdatedAt time.Time
	IsFlagged bool
}

func NewUserLocation(userID uuid.UUID, lat, lng float64) *UserLocation {
	return &UserLocation{
		UserID: userID,
		Lat:    lat,
		Lng:    lng,
	}
}

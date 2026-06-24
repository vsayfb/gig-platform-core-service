package location

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
)

const (
	maxSpeedKmH = 200.0
)

type UserLocationService struct {
	repo UserLocationRepository
}

func NewUserLocationService(repo UserLocationRepository) *UserLocationService {
	return &UserLocationService{repo: repo}
}

func (s *UserLocationService) Upsert(ctx context.Context, userID uuid.UUID, lat, lng float64) (*UserLocation, error) {
	if err := validateCoordinates(lat, lng); err != nil {
		return nil, err
	}

	existing, err := s.repo.FindByUserID(ctx, userID)
	if err != nil && !errors.Is(err, ErrLocationNotFound) {
		return nil, fmt.Errorf("failed to fetch existing location: %w", err)
	}

	// first time setting location
	if errors.Is(err, ErrLocationNotFound) {
		loc := NewUserLocation(userID, lat, lng)
		return s.repo.Save(ctx, loc)
	}

	// velocity check
	existing.IsFlagged = isSuspicious(existing, lat, lng)
	if existing.IsFlagged {
		slog.Warn("suspicious location update",
			"user_id", userID,
			"from_lat", existing.Lat,
			"from_lng", existing.Lng,
			"to_lat", lat,
			"to_lng", lng,
		)
	}

	existing.Lat = lat
	existing.Lng = lng
	return s.repo.Update(ctx, existing)
}

func (s *UserLocationService) FindNearby(ctx context.Context, requesterID uuid.UUID, lat, lng, radiusKm float64) ([]*UserLocation, error) {
	if err := validateCoordinates(lat, lng); err != nil {
		return nil, err
	}
	if radiusKm <= 0 {
		return nil, fmt.Errorf("radiusKm must be positive")
	}
	return s.repo.FindNearby(ctx, lat, lng, radiusKm, requesterID)
}

func validateCoordinates(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return fmt.Errorf("invalid latitude: %f", lat)
	}
	if lng < -180 || lng > 180 {
		return fmt.Errorf("invalid longitude: %f", lng)
	}
	return nil
}

func isSuspicious(existing *UserLocation, newLat, newLng float64) bool {
	dist := haversineKm(existing.Lat, existing.Lng, newLat, newLng)
	elapsed := time.Since(existing.UpdatedAt).Hours()
	if elapsed <= 0 {
		return dist > 0
	}
	return (dist / elapsed) > maxSpeedKmH
}

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * (math.Pi / 180)
	dLng := (lng2 - lng1) * (math.Pi / 180)
	a := math.Pow(math.Sin(dLat/2), 2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Pow(math.Sin(dLng/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

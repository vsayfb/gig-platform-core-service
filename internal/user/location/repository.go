package location

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrLocationNotFound = errors.New("location not found")

type UserLocationRepository interface {
	Save(ctx context.Context, loc *UserLocation) (*UserLocation, error)
	Update(ctx context.Context, loc *UserLocation) (*UserLocation, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) (*UserLocation, error)
}

type userLocationRepository struct {
	db *pgxpool.Pool
}

func NewUserLocationRepository(db *pgxpool.Pool) UserLocationRepository {
	return &userLocationRepository{db: db}
}

const saveUserLocationQuery = `
	INSERT INTO user_locations (user_id, location)
	VALUES ($1, ST_MakePoint($2, $3)::geography)
	RETURNING id, user_id, ST_X(location::geometry), ST_Y(location::geometry), updated_at, is_flagged
`

const updateUserLocationQuery = `
	UPDATE user_locations
	SET location = ST_MakePoint($1, $2)::geography, updated_at = NOW()
	WHERE user_id = $3
	RETURNING id, user_id, ST_X(location::geometry), ST_Y(location::geometry), updated_at, is_flagged
`

const findUserLocationByUserIDQuery = `
	SELECT id, user_id, ST_X(location::geometry), ST_Y(location::geometry), updated_at, is_flagged
	FROM user_locations
	WHERE user_id = $1
`

func (r *userLocationRepository) Save(ctx context.Context, loc *UserLocation) (*UserLocation, error) {
	row := r.db.QueryRow(ctx, saveUserLocationQuery, loc.UserID, loc.Lng, loc.Lat)

	return r.scan(row)
}

func (r *userLocationRepository) Update(ctx context.Context, loc *UserLocation) (*UserLocation, error) {
	row := r.db.QueryRow(ctx, updateUserLocationQuery, loc.Lng, loc.Lat, loc.UserID)

	return r.scan(row)
}

func (r *userLocationRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*UserLocation, error) {
	row := r.db.QueryRow(ctx, findUserLocationByUserIDQuery, userID)

	result, err := r.scan(row)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrLocationNotFound
	}

	return result, err
}

func (r *userLocationRepository) scan(row pgx.Row) (*UserLocation, error) {
	l := &UserLocation{}

	err := row.Scan(
		&l.ID,
		&l.UserID,
		&l.Lng,
		&l.Lat,
		&l.UpdatedAt,
		&l.IsFlagged,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan user location: %w", err)
	}

	return l, nil
}

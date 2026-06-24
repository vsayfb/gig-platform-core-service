package reputation

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrReputationNotFound = errors.New("reputation not found")

type UserReputationRepository interface {
	Save(ctx context.Context, rep *UserReputation) (*UserReputation, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) (*UserReputation, error)
	Update(ctx context.Context, rep *UserReputation) (*UserReputation, error)
}

type userReputationRepository struct {
	db *pgxpool.Pool
}

func NewUserReputationRepository(db *pgxpool.Pool) UserReputationRepository {
	return &userReputationRepository{db: db}
}

const saveUserReputationQuery = `
	INSERT INTO user_reputations (user_id)
	VALUES ($1)
	RETURNING id, user_id, rating_as_employer, rating_as_employee, rating_count
`

const findUserReputationByUserIDQuery = `
	SELECT id, user_id, rating_as_employer, rating_as_employee, rating_count
	FROM user_reputations
	WHERE user_id = $1
`

const updateUserReputationQuery = `
	UPDATE user_reputations
	SET rating_as_employer = $1, rating_as_employee = $2, rating_count = $3
	WHERE user_id = $4
	RETURNING id, user_id, rating_as_employer, rating_as_employee, rating_count
`

func (r *userReputationRepository) Save(ctx context.Context, rep *UserReputation) (*UserReputation, error) {
	row := r.db.QueryRow(ctx, saveUserReputationQuery, rep.UserID)
	return r.scan(row)
}

func (r *userReputationRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*UserReputation, error) {
	row := r.db.QueryRow(ctx, findUserReputationByUserIDQuery, userID)

	result, err := r.scan(row)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrReputationNotFound
	}

	return result, err
}

func (r *userReputationRepository) Update(ctx context.Context, rep *UserReputation) (*UserReputation, error) {
	row := r.db.QueryRow(ctx, updateUserReputationQuery,
		rep.RatingAsEmployer,
		rep.RatingAsEmployee,
		rep.RatingCount,
		rep.UserID,
	)

	return r.scan(row)
}

func (r *userReputationRepository) scan(row pgx.Row) (*UserReputation, error) {
	rep := &UserReputation{}

	err := row.Scan(
		&rep.ID,
		&rep.UserID,
		&rep.RatingAsEmployer,
		&rep.RatingAsEmployee,
		&rep.RatingCount,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan user reputation: %w", err)
	}

	return rep, nil
}

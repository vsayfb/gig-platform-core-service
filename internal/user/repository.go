package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	Save(ctx context.Context, user *User) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	Update(ctx context.Context, user *User) (*User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Save(ctx context.Context, user *User) (*User, error) {
	query := `
        INSERT INTO users (name, avatar_url, email, bio)
        VALUES ($1, $2, $3, $4)
        RETURNING id, name, avatar_url, email, bio,
                  is_verified, is_available_today,
                  last_active_at, created_at
    `

	row := r.db.QueryRow(
		ctx,
		query,
		user.Name,
		user.AvatarURL,
		user.Email,
		user.Bio,
	)

	return scanUser(row)
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
        SELECT id, name, avatar_url, email, bio, is_verified, is_available_today, last_active_at, created_at
        FROM users
        WHERE id = $1
    `
	row := r.db.QueryRow(ctx, query, id)

	return scanUser(row)
}

func (r *userRepository) Update(ctx context.Context, user *User) (*User, error) {
	query := `
        UPDATE users
        SET name = $1, avatar_url = $2, bio = $3, is_available_today = $4, last_active_at = $5
        WHERE id = $6
        RETURNING id, name, avatar_url, email, bio, is_verified, is_available_today, last_active_at, created_at
    `

	row := r.db.QueryRow(ctx, query,
		user.Name,
		user.AvatarURL,
		user.Bio,
		user.IsAvailableToday,
		user.LastActiveAt,
		user.ID,
	)

	return scanUser(row)
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)

	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(row scanner) (*User, error) {
	u := &User{}

	err := row.Scan(
		&u.ID,
		&u.Name,
		&u.AvatarURL,
		&u.Email,
		&u.Bio,
		&u.IsVerified,
		&u.IsAvailableToday,
		&u.LastActiveAt,
		&u.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	return u, nil
}

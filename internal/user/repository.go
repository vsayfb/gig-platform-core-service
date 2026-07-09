package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vsayfb/gig-platform-core-service/pkg/dbtx"
)

type UserRepository interface {
	Save(ctx context.Context, user *User) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	Update(ctx context.Context, user *User) (*User, error)
	Delete(ctx context.Context, id uuid.UUID) error
	InsertFCMToken(ctx context.Context, userID uuid.UUID, token string) error
	DeleteFCMToken(ctx context.Context, userID uuid.UUID, token string) error
	InsertCategories(ctx context.Context, userID uuid.UUID, categoryIDs []uuid.UUID) error
	FindSummariesByIDs(ctx context.Context, ids []uuid.UUID) ([]*UserSummary, error)
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
	row := dbtx.Extract(ctx, r.db).QueryRow(ctx, query,
		user.Name,
		user.AvatarURL,
		user.Email,
		user.Bio,
	)

	return scanUser(row)
}

func (r *userRepository) InsertCategories(ctx context.Context, userID uuid.UUID, categoryIDs []uuid.UUID) error {
	if len(categoryIDs) == 0 {
		return nil
	}

	const query = `
		INSERT INTO user_categories (user_id, category_id)
		SELECT $1, unnest($2::uuid[])
		ON CONFLICT (user_id, category_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, userID, categoryIDs)

	return err
}

func (r *userRepository) InsertFCMToken(ctx context.Context, userID uuid.UUID, token string) error {
	if token == "" {
		return nil
	}

	query := `
		INSERT INTO fcm_tokens (user_id, token, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (user_id, token) DO UPDATE 
		SET updated_at = NOW()
	`

	_, err := r.db.Exec(ctx, query, userID, token)

	if err != nil {
		return fmt.Errorf("failed to insert FCM token: %w", err)
	}

	return nil
}

func (r *userRepository) DeleteFCMToken(ctx context.Context, userID uuid.UUID, token string) error {
	if token == "" {
		return nil
	}

	query := `DELETE FROM fcm_tokens WHERE user_id = $1 AND token = $2`

	result, err := r.db.Exec(ctx, query, userID, token)

	if err != nil {
		return fmt.Errorf("failed to delete FCM token: %w", err)
	}

	rowsAffected := result.RowsAffected()

	if rowsAffected == 0 {
		return fmt.Errorf("token not found for user")
	}

	return nil
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

func (r *userRepository) FindSummariesByIDs(ctx context.Context, ids []uuid.UUID) ([]*UserSummary, error) {
	query := `
		SELECT id, name, avatar_url
		FROM users
		WHERE id = ANY($1)
	`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("query user summaries: %w", err)
	}
	defer rows.Close()

	var summaries []*UserSummary

	for rows.Next() {
		summary := &UserSummary{}

		if err := rows.Scan(
			&summary.ID,
			&summary.Name,
			&summary.AvatarURL,
		); err != nil {
			return nil, fmt.Errorf("scan user summary: %w", err)
		}

		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user summaries: %w", err)
	}

	return summaries, nil
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

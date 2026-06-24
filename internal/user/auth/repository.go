package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserAuthRepository interface {
	Save(ctx context.Context, auth *UserAuth) (*UserAuth, error)
	FindByGoogleSub(ctx context.Context, googleSub string) (*UserAuth, error)
	FindByPhoneHmac(ctx context.Context, hmac string) (*UserAuth, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) (*UserAuth, error)
}

type userAuthRepository struct {
	db *pgxpool.Pool
}

func NewUserAuthRepository(db *pgxpool.Pool) UserAuthRepository {
	return &userAuthRepository{db: db}
}

func (r *userAuthRepository) Save(ctx context.Context, auth *UserAuth) (*UserAuth, error) {
	query := `
        INSERT INTO user_auth (user_id, google_sub, phone_encrypted, phone_hmac)
        VALUES ($1, $2, $3, $4)
        RETURNING id, user_id, google_sub, phone_encrypted, phone_hmac, created_at
    `
	row := r.db.QueryRow(ctx, query,
		auth.UserID,
		auth.GoogleSub,
		auth.PhoneEncrypted,
		auth.PhoneHMAC,
	)
	return scanUserAuth(row)
}

func (r *userAuthRepository) FindByGoogleSub(ctx context.Context, googleSub string) (*UserAuth, error) {
	query := `
        SELECT id, user_id, google_sub, phone_encrypted, phone_hmac, created_at
        FROM user_auth
        WHERE google_sub = $1
    `
	row := r.db.QueryRow(ctx, query, googleSub)
	return scanUserAuth(row)
}

func (r *userAuthRepository) FindByPhoneHmac(ctx context.Context, hmac string) (*UserAuth, error) {
	query := `
        SELECT id, user_id, google_sub, phone_encrypted, phone_hmac, created_at
        FROM user_auth
        WHERE phone_hmac = $1
    `
	row := r.db.QueryRow(ctx, query, hmac)
	return scanUserAuth(row)
}

func (r *userAuthRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*UserAuth, error) {
	query := `
        SELECT id, user_id, google_sub, phone_encrypted, phone_hmac, created_at
        FROM user_auth
        WHERE user_id = $1
    `
	row := r.db.QueryRow(ctx, query, userID)
	return scanUserAuth(row)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUserAuth(row scanner) (*UserAuth, error) {
	a := &UserAuth{}
	err := row.Scan(
		&a.ID,
		&a.UserID,
		&a.GoogleSub,
		&a.PhoneEncrypted,
		&a.PhoneHMAC,
		&a.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan user auth: %w", err)
	}
	return a, nil
}

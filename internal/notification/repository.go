package notification

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) UpsertFCMToken(ctx context.Context, userID uuid.UUID, token string) error {

	const q = `
		INSERT INTO fcm_tokens (user_id, token)
		VALUES ($1, $2)
		ON CONFLICT (user_id, token)
		DO UPDATE SET updated_at = now()
	`

	if _, err := r.db.Exec(ctx, q, userID, token); err != nil {
		return fmt.Errorf("upsert fcm token: %w", err)
	}
	return nil
}

func (r *NotificationRepository) DeleteFCMToken(ctx context.Context, userID uuid.UUID, token string) error {
	const q = `DELETE FROM fcm_tokens WHERE user_id = $1 AND token = $2`

	if _, err := r.db.Exec(ctx, q, userID, token); err != nil {
		return fmt.Errorf("delete fcm token: %w", err)
	}

	return nil
}

func (r *NotificationRepository) ListFCMTokens(ctx context.Context, userID uuid.UUID) ([]string, error) {
	const q = `SELECT token FROM fcm_tokens WHERE user_id = $1`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list fcm tokens: %w", err)
	}

	defer rows.Close()

	var tokens []string

	for rows.Next() {
		var t string

		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("scan fcm token: %w", err)
		}

		tokens = append(tokens, t)
	}

	return tokens, rows.Err()
}

func (r *NotificationRepository) ListNotifications(ctx context.Context, userID uuid.UUID, p ListNotificationsParams) ([]Notification, error) {
	q := `
		SELECT id, user_id, type, ref_gig_id, title, body, is_read, created_at
		FROM notifications
		WHERE user_id = $1
	`
	args := []any{userID}

	if p.UnreadOnly {
		q += ` AND is_read = false`
	}

	q += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	args = append(args, p.Limit, p.Offset)

	rows, err := r.db.Query(ctx, q, args...)

	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}

	defer rows.Close()

	var out []Notification

	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.RefGigID, &n.Title, &n.Body, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		out = append(out, n)
	}

	return out, rows.Err()
}

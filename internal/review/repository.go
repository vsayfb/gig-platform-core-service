package review

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vsayfb/gig-platform-core-service/pkg/dbtx"
)

var (
	ErrNotFound        = errors.New("review: not found")
	ErrAlreadyReviewed = errors.New("review: already submitted a review for this contract")
)

const (
	queryInsert = `
		INSERT INTO reviews (id, contract_id, reviewer_id, reviewee_id, rating, comment, role_context, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	queryFindByContractAndReviewer = `
		SELECT id, contract_id, reviewer_id, reviewee_id, rating, comment, role_context, created_at
		FROM reviews WHERE contract_id = $1 AND reviewer_id = $2`

	queryFindByReviewee = `
		SELECT id, contract_id, reviewer_id, reviewee_id, rating, comment, role_context, created_at
		FROM reviews WHERE reviewee_id = $1
		ORDER BY created_at DESC`
)

type ReviewRepository interface {
	Save(ctx context.Context, r *Review) error
	FindByContractAndReviewer(ctx context.Context, contractID, reviewerID uuid.UUID) (*Review, error)
	FindByReviewee(ctx context.Context, revieweeID uuid.UUID) ([]*Review, error)
}

type repository struct {
	db *pgxpool.Pool
}

func NewReviewRepository(db *pgxpool.Pool) ReviewRepository {
	return &repository{db: db}
}

func (r *repository) Save(ctx context.Context, rev *Review) error {
	db := dbtx.Extract(ctx, r.db)
	_, err := db.Exec(ctx, queryInsert,
		rev.ID, rev.ContractID, rev.ReviewerID, rev.RevieweeID,
		rev.Rating, rev.Comment, rev.RoleContext, rev.CreatedAt,
	)
	return err
}

func (r *repository) FindByContractAndReviewer(ctx context.Context, contractID, reviewerID uuid.UUID) (*Review, error) {
	row := dbtx.Extract(ctx, r.db).QueryRow(ctx, queryFindByContractAndReviewer, contractID, reviewerID)

	return r.scan(row)
}

func (r *repository) FindByReviewee(ctx context.Context, revieweeID uuid.UUID) ([]*Review, error) {
	rows, err := dbtx.Extract(ctx, r.db).Query(ctx, queryFindByReviewee, revieweeID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var reviews []*Review

	for rows.Next() {
		rev, err := r.scan(rows)

		if err != nil {
			return nil, err
		}

		reviews = append(reviews, rev)
	}

	return reviews, nil
}

func (r *repository) scan(row pgx.Row) (*Review, error) {
	rev := &Review{}

	err := row.Scan(
		&rev.ID, &rev.ContractID, &rev.ReviewerID, &rev.RevieweeID,
		&rev.Rating, &rev.Comment, &rev.RoleContext, &rev.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return rev, nil
}

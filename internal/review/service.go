package review

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vsayfb/gig-platform-core-service/internal/contract"
	"github.com/vsayfb/gig-platform-core-service/internal/user/reputation"
	"github.com/vsayfb/gig-platform-core-service/pkg/dbtx"
)

var (
	ErrNotParty        = errors.New("review: caller is not a party to this contract")
	ErrContractNotDone = errors.New("review: contract is not completed")
	ErrInvalidRating   = errors.New("review: rating must be between 1 and 5")
)

type ReviewService interface {
	Submit(ctx context.Context, contractID uuid.UUID, reviewerID uuid.UUID, in CreateReviewInput) (*Review, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*Review, error)
}

type service struct {
	repo         ReviewRepository
	contractRepo contract.ContractRepository
	repuService  reputation.UserReputationService
	db           *pgxpool.Pool
}

func NewReviewService(
	repo ReviewRepository,
	contractRepo contract.ContractRepository,
	repuService reputation.UserReputationService,
	db *pgxpool.Pool,
) ReviewService {
	return &service{repo: repo, contractRepo: contractRepo, repuService: repuService, db: db}
}

func (s *service) Submit(ctx context.Context, contractID uuid.UUID, reviewerID uuid.UUID, in CreateReviewInput) (*Review, error) {
	if in.Rating < 1 || in.Rating > 5 {
		return nil, ErrInvalidRating
	}

	c, err := s.contractRepo.FindByID(ctx, contractID)

	if err != nil {
		return nil, err
	}

	if c.Status != contract.StatusCompleted {
		return nil, ErrContractNotDone
	}

	// Determine reviewer role and reviewee.
	var revieweeID uuid.UUID
	var roleCtx RoleContext

	switch reviewerID {
	case c.EmployerID:
		revieweeID = c.EmployeeID
		roleCtx = RoleAsEmployer
	case c.EmployeeID:
		revieweeID = c.EmployerID
		roleCtx = RoleAsEmployee
	default:
		return nil, ErrNotParty
	}

	// Duplicate check.
	existing, err := s.repo.FindByContractAndReviewer(ctx, contractID, reviewerID)

	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	if existing != nil {
		return nil, ErrAlreadyReviewed
	}

	rev := &Review{
		ID:          uuid.New(),
		ContractID:  contractID,
		ReviewerID:  reviewerID,
		RevieweeID:  revieweeID,
		Rating:      in.Rating,
		Comment:     in.Comment,
		RoleContext: roleCtx,
		CreatedAt:   time.Now().UTC(),
	}

	err = dbtx.RunInTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.repo.Save(ctx, rev); err != nil {
			return err
		}

		// Recalculate reviewee's reputation.
		return s.repuService.Recalculate(ctx, revieweeID, float32(in.Rating), roleCtx == RoleAsEmployee)
	})

	if err != nil {
		slog.ErrorContext(ctx, "review.Submit: transaction failed", "err", err)
		return nil, err
	}

	return rev, nil
}

func (s *service) ListByUser(ctx context.Context, userID uuid.UUID) ([]*Review, error) {
	return s.repo.FindByReviewee(ctx, userID)
}

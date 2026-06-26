package application

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/vsayfb/gig-platform-core-service/internal/gig"
)

var (
	ErrGigNotOpen      = errors.New("application: gig is not open")
	ErrCannotApplyOwn  = errors.New("application: cannot apply to your own gig")
	ErrNotWithdrawable = errors.New("application: only PENDING applications can be withdrawn")
)

type ApplicationService interface {
	ListByGig(ctx context.Context, gigID uuid.UUID, callerID uuid.UUID) ([]*Application, error)
	Get(ctx context.Context, id uuid.UUID, callerID uuid.UUID) (*Application, error)
	Apply(ctx context.Context, gigID uuid.UUID, applicantID uuid.UUID) (*Application, error)
	Withdraw(ctx context.Context, id uuid.UUID, applicantID uuid.UUID) error
}

type service struct {
	repo    ApplicationRepository
	gigRepo gig.GigRepository
}

func NewApplicationService(repo ApplicationRepository, gigRepo gig.GigRepository) ApplicationService {
	return &service{repo: repo, gigRepo: gigRepo}
}

func (s *service) ListByGig(ctx context.Context, gigID uuid.UUID, callerID uuid.UUID) ([]*Application, error) {
	detail, err := s.gigRepo.FindByID(ctx, gigID)

	if err != nil {
		return nil, err
	}

	if detail.Gig.PosterID != callerID {
		return nil, gig.ErrNotPoster
	}

	return s.repo.FindByGigID(ctx, gigID)
}

func (s *service) Get(ctx context.Context, id uuid.UUID, callerID uuid.UUID) (*Application, error) {
	a, err := s.repo.FindByID(ctx, id)

	if err != nil {
		return nil, err
	}

	// Visible to the applicant or the gig poster.
	if a.ApplicantID != callerID {
		detail, err := s.gigRepo.FindByID(ctx, a.GigID)
		if err != nil {
			return nil, err
		}
		if detail.Gig.PosterID != callerID {
			return nil, ErrNotApplicant
		}
	}

	return a, nil
}

func (s *service) Apply(ctx context.Context, gigID uuid.UUID, applicantID uuid.UUID) (*Application, error) {
	detail, err := s.gigRepo.FindByID(ctx, gigID)

	if err != nil {
		return nil, err
	}

	if detail.Gig.Status != gig.StatusOpen {
		return nil, ErrGigNotOpen
	}

	if detail.Gig.PosterID == applicantID {
		return nil, ErrCannotApplyOwn
	}

	// Duplicate check.
	existing, err := s.repo.FindByGigAndApplicant(ctx, gigID, applicantID)

	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	if existing != nil {
		return nil, ErrAlreadyApplied
	}

	a := &Application{
		ID:          uuid.New(),
		GigID:       gigID,
		ApplicantID: applicantID,
		Status:      StatusPending,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.repo.Save(ctx, a); err != nil {
		slog.ErrorContext(ctx, "application.Apply: save failed", "err", err)

		return nil, err
	}

	return a, nil
}

func (s *service) Withdraw(ctx context.Context, id uuid.UUID, applicantID uuid.UUID) error {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if a.ApplicantID != applicantID {
		return ErrNotApplicant
	}

	if a.Status != StatusPending {
		return ErrNotWithdrawable
	}

	return s.repo.UpdateStatus(ctx, id, StatusWithdrawn)
}

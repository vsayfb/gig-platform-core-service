package gig

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

var (
	ErrGigNotEditable      = errors.New("gig: only OPEN gigs can be edited")
	ErrGigNotCancellable   = errors.New("gig: only OPEN or IN_PROGRESS gigs can be cancelled")
	ErrInvalidInput        = errors.New("gig: invalid input")
	ErrLocationNotProvided = errors.New("gig: location is required")
	ErrEndDateRequired     = errors.New("gig: end_date is required when start_date is provided")
	ErrCurrencyRequired    = errors.New("gig: pay_currency is required when pay_amount is provided")
)

type GigService struct {
	repo GigRepository
}

func NewGigService(repo GigRepository) *GigService {
	return &GigService{repo: repo}
}

func (s *GigService) Feed(ctx context.Context, p FeedParams) ([]*GigFull, error) {
	if p.RadiusMeters <= 0 {
		p.RadiusMeters = 5000
	}
	if p.Limit <= 0 || p.Limit > 50 {
		p.Limit = 20
	}
	return s.repo.FindFeed(ctx, p)
}

func (s *GigService) Get(ctx context.Context, id uuid.UUID) (*GigFull, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *GigService) Create(ctx context.Context, posterID uuid.UUID, in CreateGigInput) (*GigFull, error) {
	if err := validateCreate(in); err != nil {
		return nil, err
	}

	g := &Gig{
		ID:               uuid.New(),
		PosterID:         posterID,
		Title:            in.Title,
		DescriptionRaw:   in.DescriptionRaw,
		DescriptionClean: in.DescriptionClean,
		Status:           StatusOpen,
		CreatedAt:        time.Now().UTC(),
	}

	var details *GigDetails

	if in.DurationType != nil || in.StartDate != nil || in.EndDate != nil ||
		in.PayAmount != nil || in.PayCurrency != nil || in.ExpiresAt != nil {
		details = &GigDetails{
			GigID:        g.ID,
			DurationType: in.DurationType,
			StartDate:    in.StartDate,
			EndDate:      in.EndDate,
			PayAmount:    in.PayAmount,
			PayCurrency:  in.PayCurrency,
			ExpiresAt:    in.ExpiresAt,
		}
	}

	loc := &GigLocation{
		ID:       uuid.New(),
		GigID:    g.ID,
		Lat:      *in.Lat,
		Lng:      *in.Lng,
		City:     *in.City,
		District: *in.District,
	}

	if in.ExpiresAt == nil {
		t := time.Now().UTC().AddDate(0, 1, 0)
		in.ExpiresAt = &t
	}

	if err := s.repo.Save(ctx, g, details, loc, in.CategoryIDs); err != nil {
		slog.ErrorContext(ctx, "gig.Create: save failed", "err", err)
		return nil, err
	}

	return s.repo.FindByID(ctx, g.ID)
}

func (s *GigService) Edit(ctx context.Context, gigID uuid.UUID, posterID uuid.UUID, in UpdateGigInput) (*GigFull, error) {
	full, err := s.repo.FindByID(ctx, gigID)
	if err != nil {
		return nil, err
	}
	if full.Gig.Status != StatusOpen {
		return nil, ErrGigNotEditable
	}

	if err := s.repo.Update(ctx, gigID, posterID, in); err != nil {
		return nil, err
	}

	return s.repo.FindByID(ctx, gigID)
}

func (s *GigService) Cancel(ctx context.Context, gigID uuid.UUID, callerID uuid.UUID) error {
	full, err := s.repo.FindByID(ctx, gigID)
	if err != nil {
		return err
	}
	if full.Gig.PosterID != callerID {
		return ErrNotPoster
	}
	if full.Gig.Status != StatusOpen && full.Gig.Status != StatusInProgress {
		return ErrGigNotCancellable
	}
	return s.repo.UpdateStatus(ctx, gigID, StatusCancelled)
}

func validateCreate(in CreateGigInput) error {
	if in.Title == "" || in.DescriptionRaw == "" {
		return ErrInvalidInput
	}
	if !validateLocation(in) {
		return ErrLocationNotProvided
	}
	if in.StartDate != nil && in.EndDate == nil {
		return ErrEndDateRequired
	}
	if in.PayAmount != nil && in.PayCurrency == nil {
		return ErrCurrencyRequired
	}
	return nil
}

func validateLocation(in CreateGigInput) bool {
	if in.Lat == nil || in.Lng == nil || in.City == nil || in.District == nil {
		return false
	}
	return true
}

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

type GigService interface {
	Feed(ctx context.Context, p FeedParams) ([]*GigFull, error)
	Get(ctx context.Context, id uuid.UUID) (*GigFull, error)
	Create(ctx context.Context, posterID uuid.UUID, in CreateGigInput) (*GigFull, error)
	Edit(ctx context.Context, gigID uuid.UUID, posterID uuid.UUID, in UpdateGigInput) (*GigFull, error)
	Cancel(ctx context.Context, gigID uuid.UUID, callerID uuid.UUID) error
}

type service struct {
	repo GigRepository
}

func NewGigService(repo GigRepository) GigService {
	return &service{repo: repo}
}

func (s *service) Feed(ctx context.Context, p FeedParams) ([]*GigFull, error) {
	if p.RadiusMeters <= 0 {
		p.RadiusMeters = RADIUS_METERS
	}

	if p.Limit <= 0 || p.Limit > 50 {
		p.Limit = 20
	}

	return s.repo.FindFeed(ctx, p)
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (*GigFull, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, posterID uuid.UUID, in CreateGigInput) (*GigFull, error) {
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

	loc := &GigLocation{
		ID:    uuid.New(),
		GigID: g.ID,
		Lat:   *in.Location.Lat,
		Lng:   *in.Location.Lng,
		City:  *in.Location.City,
	}

	t := time.Now().UTC().AddDate(0, 1, 0)

	details = &GigDetails{
		GigID:     g.ID,
		ExpiresAt: &t,
	}

	if err := s.repo.Save(ctx, g, details, loc); err != nil {
		slog.ErrorContext(ctx, "gig.Create: save failed", "err", err)
		return nil, err
	}

	return s.repo.FindByID(ctx, g.ID)
}

func (s *service) Edit(ctx context.Context, gigID uuid.UUID, posterID uuid.UUID, in UpdateGigInput) (*GigFull, error) {
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

func (s *service) Cancel(ctx context.Context, gigID uuid.UUID, callerID uuid.UUID) error {
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

	if !validateLocation(*in.Location) {
		return ErrLocationNotProvided
	}

	return nil
}

func validateLocation(in CreateGigInputLoc) bool {
	if in.Lat == nil || in.Lng == nil || in.City == nil {
		return false
	}
	return true
}

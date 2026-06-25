package gig

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

var (
	ErrGigNotEditable    = errors.New("gig: only OPEN gigs can be edited")
	ErrGigNotCancellable = errors.New("gig: only OPEN or IN_PROGRESS gigs can be cancelled")
	ErrInvalidInput      = errors.New("gig: invalid input")
)

type Service interface {
	Feed(ctx context.Context, p FeedParams) ([]*GigDetail, error)
	Get(ctx context.Context, id uuid.UUID) (*GigDetail, error)
	Create(ctx context.Context, posterID uuid.UUID, in CreateGigInput) (*GigDetail, error)
	Edit(ctx context.Context, gigID uuid.UUID, posterID uuid.UUID, in UpdateGigInput) (*GigDetail, error)
	Cancel(ctx context.Context, gigID uuid.UUID, callerID uuid.UUID) error
}

type GigService struct {
	repo GigRepository
}

func NewGigService(repo GigRepository) Service {
	return &GigService{repo: repo}
}

func (s *GigService) Feed(ctx context.Context, p FeedParams) ([]*GigDetail, error) {
	if p.RadiusMeters <= 0 {
		p.RadiusMeters = 50000 // default 50km
	}

	if p.Limit <= 0 || p.Limit > 50 {
		p.Limit = 20
	}

	return s.repo.FindFeed(ctx, p)
}

func (s *GigService) Get(ctx context.Context, id uuid.UUID) (*GigDetail, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *GigService) Create(ctx context.Context, posterID uuid.UUID, in CreateGigInput) (*GigDetail, error) {
	if err := s.validateCreate(in); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	g := &Gig{
		ID:               uuid.New(),
		PosterID:         posterID,
		Title:            in.Title,
		DescriptionRaw:   in.DescriptionRaw,
		DescriptionClean: in.DescriptionClean,
		DurationType:     in.DurationType,
		StartDate:        in.StartDate,
		EndDate:          in.EndDate,
		Slots:            in.Slots,
		Status:           StatusOpen,
		CreatedAt:        now,
		ExpiresAt:        in.ExpiresAt,
	}

	if g.Slots < 1 {
		g.Slots = 1
	}

	loc := &GigLocation{
		ID:       uuid.New(),
		GigID:    g.ID,
		Lat:      in.Lat,
		Lng:      in.Lng,
		City:     in.City,
		District: in.District,
	}

	if err := s.repo.Save(ctx, g, loc, in.CategoryIDs); err != nil {
		slog.ErrorContext(ctx, "gig.Create: save failed", "err", err)
		return nil, err
	}

	return s.repo.FindByID(ctx, g.ID)
}

func (s *GigService) Edit(ctx context.Context, gigID uuid.UUID, posterID uuid.UUID, in UpdateGigInput) (*GigDetail, error) {
	detail, err := s.repo.FindByID(ctx, gigID)
	if err != nil {
		return nil, err
	}

	if detail.Gig.Status != StatusOpen {
		return nil, ErrGigNotEditable
	}

	if err := s.repo.Update(ctx, gigID, posterID, in); err != nil {
		return nil, err
	}

	return s.repo.FindByID(ctx, gigID)
}

func (s *GigService) Cancel(ctx context.Context, gigID uuid.UUID, callerID uuid.UUID) error {
	detail, err := s.repo.FindByID(ctx, gigID)

	if err != nil {
		return err
	}

	if detail.Gig.PosterID != callerID {
		return ErrNotPoster
	}

	if detail.Gig.Status != StatusOpen && detail.Gig.Status != StatusInProgress {
		return ErrGigNotCancellable
	}

	return s.repo.UpdateStatus(ctx, gigID, StatusCancelled)
}

func (s *GigService) validateCreate(in CreateGigInput) error {
	if in.Title == "" {
		return ErrInvalidInput
	}

	if in.DescriptionRaw == "" {
		return ErrInvalidInput
	}

	if in.DurationType != DurationDaily && in.DurationType != DurationWeekly && in.DurationType != DurationMonthly {
		return ErrInvalidInput
	}

	if in.Lat == 0 && in.Lng == 0 {
		return ErrInvalidInput
	}

	return nil
}

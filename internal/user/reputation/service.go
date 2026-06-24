package reputation

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type UserReputationService struct {
	repo UserReputationRepository
}

func NewUserReputationService(repo UserReputationRepository) *UserReputationService {
	return &UserReputationService{repo: repo}
}

func (s *UserReputationService) Initialize(ctx context.Context, userID uuid.UUID) (*UserReputation, error) {
	rep := NewUserReputation(userID)

	created, err := s.repo.Save(ctx, rep)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize reputation: %w", err)
	}

	return created, nil
}

func (s *UserReputationService) FindByUserID(ctx context.Context, userID uuid.UUID) (*UserReputation, error) {
	rep, err := s.repo.FindByUserID(ctx, userID)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch reputation: %w", err)
	}

	return rep, nil
}

func (s *UserReputationService) Recalculate(ctx context.Context, userID uuid.UUID, newRating float32, asEmployer bool) error {
	rep, err := s.repo.FindByUserID(ctx, userID)

	if err != nil {
		return fmt.Errorf("failed to fetch reputation: %w", err)
	}

	if asEmployer {
		rep.RatingAsEmployer = recalculateAverage(rep.RatingAsEmployer, rep.RatingCount, newRating)
	} else {
		rep.RatingAsEmployee = recalculateAverage(rep.RatingAsEmployee, rep.RatingCount, newRating)
	}

	rep.RatingCount++

	if _, err := s.repo.Update(ctx, rep); err != nil {
		return fmt.Errorf("failed to update reputation: %w", err)
	}

	return nil
}

func recalculateAverage(current float32, count int, newRating float32) float32 {
	return (current*float32(count) + newRating) / float32(count+1)
}

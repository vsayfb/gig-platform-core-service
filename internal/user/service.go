package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrUserNotFound = errors.New("user not found")

type UserService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetSummaries(ctx context.Context, ids []uuid.UUID) ([]*UserSummary, error)
	UpdateProfile(ctx context.Context, user *User) (*User, error)
	AddFCMToken(ctx context.Context, userID uuid.UUID, token string) error
	RemoveFCMToken(ctx context.Context, userID uuid.UUID, token string) error
	PutCategories(ctx context.Context, userID uuid.UUID, categoryIDs []uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	userRepo UserRepository
}

func NewUserService(repo UserRepository) UserService {
	return &service{userRepo: repo}
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := s.userRepo.FindByID(ctx, id)

	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

func (s *service) GetSummaries(ctx context.Context, ids []uuid.UUID) ([]*UserSummary, error) {
	return s.userRepo.FindSummariesByIDs(ctx, ids)
}

func (s *service) UpdateProfile(ctx context.Context, user *User) (*User, error) {
	existing, err := s.userRepo.FindByID(ctx, user.ID)

	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	existing.Name = user.Name
	existing.Bio = user.Bio
	existing.AvatarURL = user.AvatarURL
	existing.IsAvailableToday = user.IsAvailableToday

	updated, err := s.userRepo.Update(ctx, existing)

	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return updated, nil
}

func (s *service) AddFCMToken(ctx context.Context, userID uuid.UUID, token string) error {
	err := s.userRepo.InsertFCMToken(ctx, userID, token)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (s *service) RemoveFCMToken(ctx context.Context, userID uuid.UUID, token string) error {
	err := s.userRepo.DeleteFCMToken(ctx, userID, token)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (s *service) PutCategories(ctx context.Context, userID uuid.UUID, categoryIDs []uuid.UUID) error {
	if err := s.userRepo.InsertCategories(ctx, userID, categoryIDs); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.userRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

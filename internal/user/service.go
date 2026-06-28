package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrUserNotFound = errors.New("user not found")

type UserService struct {
	userRepo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{userRepo: repo}
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := s.userRepo.FindByID(ctx, id)

	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

func (s *UserService) GetSummaries(ctx context.Context, ids []uuid.UUID) ([]*UserSummary, error) {
	return s.userRepo.FindSummariesByIDs(ctx, ids)
}

func (s *UserService) UpdateProfile(ctx context.Context, user *User) (*User, error) {
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

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.userRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

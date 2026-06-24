package category

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

var ErrCategoryNotFound = errors.New("category not found")
var ErrCategoryAlreadyExists = errors.New("category already exists")
var ErrSuggestionLimitReached = errors.New("suggestion limit reached")

type CategoryService struct {
	repo CategoryRepository
}

func NewCategoryService(repo CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) ListActive(ctx context.Context) ([]*Category, error) {
	return s.repo.FindAllActive(ctx)
}

func (s *CategoryService) ListPending(ctx context.Context) ([]*Category, error) {
	return s.repo.FindAllPending(ctx)
}

func (s *CategoryService) Suggest(ctx context.Context, name, slug string) (*Category, error) {
	existing, err := s.repo.FindBySlug(ctx, slug)

	if err == nil && existing != nil {
		return nil, ErrCategoryAlreadyExists
	}

	category := NewCategory(name, slug)
	created, err := s.repo.Save(ctx, category)

	if err != nil {
		return nil, fmt.Errorf("failed to save category suggestion: %w", err)
	}

	slog.Info("category suggested", "name", name, "slug", slug)

	return created, nil
}

func (s *CategoryService) Approve(ctx context.Context, id uuid.UUID) (*Category, error) {
	updated, err := s.repo.UpdateStatus(ctx, id, StatusActive)

	if err != nil {
		return nil, fmt.Errorf("failed to approve category: %w", err)
	}

	slog.Info("category approved", "id", id)

	return updated, nil
}

func (s *CategoryService) Reject(ctx context.Context, id uuid.UUID) (*Category, error) {
	updated, err := s.repo.UpdateStatus(ctx, id, StatusRejected)

	if err != nil {
		return nil, fmt.Errorf("failed to reject category: %w", err)
	}

	slog.Info("category rejected", "id", id)

	return updated, nil
}

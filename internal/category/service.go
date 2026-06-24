package category

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

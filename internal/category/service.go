package category

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

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

func (s *CategoryService) ListActive(
	ctx context.Context,
	cursor uuid.UUID,
	limit int,
) ([]*Category, error) {

	if limit <= 0 {
		limit = 20
	}

	if limit > 100 {
		limit = 100
	}

	categories, err := s.repo.FindAll(
		ctx,
		cursor,
		limit,
	)

	if err != nil {
		return nil, err
	}

	return categories, nil
}

func (s *CategoryService) ListBySlug(
	ctx context.Context,
	query string,
) ([]*Category, error) {

	query = strings.TrimSpace(query)

	if query == "" {
		return []*Category{}, nil
	}

	query = strings.ToLower(query)

	categories, err := s.repo.FindBySlug(
		ctx,
		query,
	)

	if err != nil {
		return nil, err
	}

	return categories, nil
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

package category

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryRepository interface {
	Save(ctx context.Context, category *Category) (*Category, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Category, error)
	FindBySlug(ctx context.Context, slug string) (*Category, error)
	FindAllActive(ctx context.Context) ([]*Category, error)
	FindAllPending(ctx context.Context) ([]*Category, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) (*Category, error)
}

type categoryRepository struct {
	db *pgxpool.Pool
}

func NewCategoryRepository(db *pgxpool.Pool) CategoryRepository {
	return &categoryRepository{db: db}
}

const saveCategoryQuery = `
	INSERT INTO categories (name, slug)
	VALUES ($1, $2)
	RETURNING id, name, slug, status, created_at
`

const findCategoryByIDQuery = `
	SELECT id, name, slug, status, created_at
	FROM categories
	WHERE id = $1
`

const findCategoryBySlugQuery = `
	SELECT id, name, slug, status, created_at
	FROM categories
	WHERE slug = $1
`

const findAllActiveCategoriesQuery = `
	SELECT id, name, slug, status, created_at
	FROM categories
	WHERE status = 'ACTIVE'
	ORDER BY name ASC
`

const findAllPendingCategoriesQuery = `
	SELECT id, name, slug, status, created_at
	FROM categories
	WHERE status = 'PENDING'
	ORDER BY created_at ASC
`

const updateCategoryStatusQuery = `
	UPDATE categories
	SET status = $1
	WHERE id = $2
	RETURNING id, name, slug, status, created_at
`

func (r *categoryRepository) Save(ctx context.Context, category *Category) (*Category, error) {
	row := r.db.QueryRow(ctx, saveCategoryQuery, category.Name, category.Slug)

	return r.scan(row)
}

func (r *categoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*Category, error) {
	row := r.db.QueryRow(ctx, findCategoryByIDQuery, id)

	return r.scan(row)
}

func (r *categoryRepository) FindBySlug(ctx context.Context, slug string) (*Category, error) {
	row := r.db.QueryRow(ctx, findCategoryBySlugQuery, slug)

	return r.scan(row)
}

func (r *categoryRepository) FindAllActive(ctx context.Context) ([]*Category, error) {
	rows, err := r.db.Query(ctx, findAllActiveCategoriesQuery)

	if err != nil {
		return nil, fmt.Errorf("failed to query active categories: %w", err)
	}

	defer rows.Close()

	return r.scanRows(rows)
}

func (r *categoryRepository) FindAllPending(ctx context.Context) ([]*Category, error) {
	rows, err := r.db.Query(ctx, findAllPendingCategoriesQuery)

	if err != nil {
		return nil, fmt.Errorf("failed to query pending categories: %w", err)
	}

	defer rows.Close()

	return r.scanRows(rows)
}

func (r *categoryRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) (*Category, error) {
	row := r.db.QueryRow(ctx, updateCategoryStatusQuery, status, id)

	return r.scan(row)
}

func (r *categoryRepository) scan(row pgx.Row) (*Category, error) {
	c := &Category{}

	err := row.Scan(
		&c.ID,
		&c.Name,
		&c.Slug,
		&c.Status,
		&c.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan category: %w", err)
	}

	return c, nil
}

func (r *categoryRepository) scanRows(rows pgx.Rows) ([]*Category, error) {
	var categories []*Category

	for rows.Next() {
		c := &Category{}
		err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.Slug,
			&c.Status,
			&c.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan category row: %w", err)
		}

		categories = append(categories, c)
	}

	return categories, nil
}

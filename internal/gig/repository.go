package gig

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vsayfb/gig-platform-core-service/pkg/dbtx"
)

var (
	ErrGigNotFound = errors.New("gig: not found")
	ErrNotPoster   = errors.New("gig: requester is not the poster")
)

const (
	queryInsertGig = `
		INSERT INTO gigs (
			id, poster_id, title, description_raw, description_clean,
			duration_type, start_date, end_date, slots, status,
			created_at, expires_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12
		)`

	queryInsertGigLocation = `
		INSERT INTO gig_locations (id, gig_id, location, city, district)
		VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, $5, $6)`

	queryInsertGigCategory = `
		INSERT INTO gig_categories (gig_id, category_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`

	queryFindByID = `
		SELECT id, poster_id, title, description_raw, description_clean,
		       duration_type, start_date, end_date, slots, status,
		       created_at, expires_at
		FROM gigs WHERE id = $1`

	queryFindLocationByGigID = `
		SELECT id, gig_id,
		       ST_Y(location::geometry) AS lat,
		       ST_X(location::geometry) AS lng,
		       city, district
		FROM gig_locations WHERE gig_id = $1`

	queryFindCategoriesByGigID = `
		SELECT category_id FROM gig_categories WHERE gig_id = $1`

	queryUpdateGigStatus = `
		UPDATE gigs SET status = $2 WHERE id = $1`

	queryUpdateGig = `
		UPDATE gigs SET
			title             = COALESCE($2, title),
			description_raw   = COALESCE($3, description_raw),
			description_clean = COALESCE($4, description_clean),
			duration_type     = COALESCE($5, duration_type),
			start_date        = COALESCE($6, start_date),
			end_date          = COALESCE($7, end_date),
			slots             = COALESCE($8, slots),
			expires_at        = COALESCE($9, expires_at)
		WHERE id = $1 AND status = 'OPEN'`

	// Feed query: location-sorted, keyset-paginated, with optional filters.
	// Filters applied dynamically in FindFeed.
	queryFeedBase = `
		SELECT g.id, g.poster_id, g.title, g.description_raw, g.description_clean,
		       g.duration_type, g.start_date, g.end_date, g.slots, g.status,
		       g.created_at, g.expires_at,
		       gl.id AS loc_id,
		       ST_Y(gl.location::geometry) AS lat,
		       ST_X(gl.location::geometry) AS lng,
		       gl.city, gl.district,
		       ST_Distance(gl.location, ST_SetSRID(ST_MakePoint($2,$1),4326)::geography) AS distance_m
		FROM gigs g
		JOIN gig_locations gl ON gl.gig_id = g.id
		WHERE g.status = 'OPEN'
		  AND ST_DWithin(gl.location, ST_SetSRID(ST_MakePoint($2,$1),4326)::geography, $3)`
)

type GigRepository interface {
	Save(ctx context.Context, g *Gig, loc *GigLocation, categoryIDs []uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*GigDetail, error)
	Update(ctx context.Context, id uuid.UUID, posterID uuid.UUID, in UpdateGigInput) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status GigStatus) error
	FindFeed(ctx context.Context, p FeedParams) ([]*GigDetail, error)
}

type repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) GigRepository {
	return &repository{db: db}
}

func (r *repository) Save(ctx context.Context, g *Gig, loc *GigLocation, categoryIDs []uuid.UUID) error {
	db := dbtx.Extract(ctx, r.db)

	_, err := db.Exec(ctx, queryInsertGig,
		g.ID, g.PosterID, g.Title, g.DescriptionRaw, g.DescriptionClean,
		g.DurationType, g.StartDate, g.EndDate, g.Slots, g.Status,
		g.CreatedAt, g.ExpiresAt,
	)

	if err != nil {
		return err
	}

	_, err = db.Exec(ctx, queryInsertGigLocation,
		loc.ID, loc.GigID, loc.Lng, loc.Lat, loc.City, loc.District,
	)

	if err != nil {
		return err
	}

	for _, catID := range categoryIDs {
		if _, err = db.Exec(ctx, queryInsertGigCategory, g.ID, catID); err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*GigDetail, error) {
	db := dbtx.Extract(ctx, r.db)

	row := db.QueryRow(ctx, queryFindByID, id)

	g, err := r.scan(row)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGigNotFound
		}
		return nil, err
	}

	detail := &GigDetail{Gig: *g}

	// location
	locRow := db.QueryRow(ctx, queryFindLocationByGigID, id)

	loc := &GigLocation{}

	err = locRow.Scan(&loc.ID, &loc.GigID, &loc.Lat, &loc.Lng, &loc.City, &loc.District)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if err == nil {
		detail.Location = loc
	}

	// categories
	rows, err := db.Query(ctx, queryFindCategoriesByGigID, id)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var catID uuid.UUID
		if err := rows.Scan(&catID); err != nil {
			return nil, err
		}
		detail.Categories = append(detail.Categories, catID)
	}

	return detail, nil
}

func (r *repository) Update(ctx context.Context, id uuid.UUID, posterID uuid.UUID, in UpdateGigInput) error {
	db := dbtx.Extract(ctx, r.db)

	// Verify ownership first.
	var storedPosterID uuid.UUID
	err := db.QueryRow(ctx, `SELECT poster_id FROM gigs WHERE id = $1`, id).Scan(&storedPosterID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrGigNotFound
		}
		return err
	}
	if storedPosterID != posterID {
		return ErrNotPoster
	}

	_, err = db.Exec(ctx, queryUpdateGig,
		id,
		in.Title,
		in.DescriptionRaw,
		in.DescriptionClean,
		in.DurationType,
		in.StartDate,
		in.EndDate,
		in.Slots,
		in.ExpiresAt,
	)
	return err
}

func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status GigStatus) error {
	db := dbtx.Extract(ctx, r.db)

	_, err := db.Exec(ctx, queryUpdateGigStatus, id, status)

	return err
}

func (r *repository) FindFeed(ctx context.Context, p FeedParams) ([]*GigDetail, error) {
	db := dbtx.Extract(ctx, r.db)

	// Build query dynamically on top of base.
	q := queryFeedBase

	args := []any{p.Lat, p.Lng, p.RadiusMeters}
	argIdx := 4

	if p.DurationType != nil {
		q += ` AND g.duration_type = $` + itoa(argIdx)
		args = append(args, *p.DurationType)
		argIdx++
	}

	if p.CategoryID != nil {
		q += ` AND EXISTS (
			SELECT 1 FROM gig_categories gc
			WHERE gc.gig_id = g.id AND gc.category_id = $` + itoa(argIdx) + `)`
		args = append(args, *p.CategoryID)
		argIdx++
	}

	if p.Cursor != nil {
		q += ` AND g.created_at < $` + itoa(argIdx)
		args = append(args, *p.Cursor)
		argIdx++
	}

	_ = argIdx

	q += ` ORDER BY distance_m ASC, g.created_at DESC LIMIT $` + itoa(len(args)+1)

	args = append(args, p.Limit)

	rows, err := db.Query(ctx, q, args...)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var feed []*GigDetail

	for rows.Next() {
		g := &Gig{}

		loc := &GigLocation{}

		var distanceM float64

		err := rows.Scan(
			&g.ID, &g.PosterID, &g.Title, &g.DescriptionRaw, &g.DescriptionClean,
			&g.DurationType, &g.StartDate, &g.EndDate, &g.Slots, &g.Status,
			&g.CreatedAt, &g.ExpiresAt,
			&loc.ID, &loc.Lat, &loc.Lng, &loc.City, &loc.District,
			&distanceM,
		)

		if err != nil {
			return nil, err
		}

		loc.GigID = g.ID

		feed = append(feed, &GigDetail{Gig: *g, Location: loc})
	}

	// Fetch categories for each gig in feed (batch would be better at scale, fine for now).
	for _, detail := range feed {
		catRows, err := db.Query(ctx, queryFindCategoriesByGigID, detail.Gig.ID)
		if err != nil {
			return nil, err
		}
		for catRows.Next() {
			var catID uuid.UUID
			if err := catRows.Scan(&catID); err != nil {
				catRows.Close()
				return nil, err
			}
			detail.Categories = append(detail.Categories, catID)
		}
		catRows.Close()
	}

	return feed, nil
}

func (r *repository) scan(row pgx.Row) (*Gig, error) {
	g := &Gig{}
	err := row.Scan(
		&g.ID, &g.PosterID, &g.Title, &g.DescriptionRaw, &g.DescriptionClean,
		&g.DurationType, &g.StartDate, &g.EndDate, &g.Slots, &g.Status,
		&g.CreatedAt, &g.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	return g, nil
}

// itoa converts int to string for query building (avoids fmt import).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte(n%10) + '0'
		n /= 10
	}
	return string(buf[pos:])
}

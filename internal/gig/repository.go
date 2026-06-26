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
		INSERT INTO gigs (id, poster_id, title, description_raw, description_clean, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	queryInsertGigDetails = `
		INSERT INTO gig_details (gig_id, duration_type, start_date, end_date, pay_amount, pay_currency, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	queryInsertGigLocation = `
		INSERT INTO gig_locations (id, gig_id, location, city, district)
		VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, $5, $6)`

	queryInsertGigCategory = `
		INSERT INTO gig_categories (gig_id, category_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`

	queryFindByID = `
		SELECT id, poster_id, title, description_raw, description_clean, status, created_at
		FROM gigs WHERE id = $1`

	queryFindDetailsByGigID = `
		SELECT gig_id, duration_type, start_date, end_date, pay_amount, pay_currency, expires_at
		FROM gig_details WHERE gig_id = $1`

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
			description_clean = COALESCE($4, description_clean)
		WHERE id = $1 AND status = 'OPEN'`

	queryUpsertGigDetails = `
		INSERT INTO gig_details (gig_id, duration_type, start_date, end_date, pay_amount, pay_currency, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (gig_id) DO UPDATE SET
			duration_type = COALESCE(EXCLUDED.duration_type, gig_details.duration_type),
			start_date    = COALESCE(EXCLUDED.start_date,    gig_details.start_date),
			end_date      = COALESCE(EXCLUDED.end_date,      gig_details.end_date),
			pay_amount    = COALESCE(EXCLUDED.pay_amount,    gig_details.pay_amount),
			pay_currency  = COALESCE(EXCLUDED.pay_currency,  gig_details.pay_currency),
			expires_at    = COALESCE(EXCLUDED.expires_at,    gig_details.expires_at)`

	queryFeedBase = `
		SELECT g.id, g.poster_id, g.title, g.description_raw, g.description_clean, g.status, g.created_at,
		       d.gig_id, d.duration_type, d.start_date, d.end_date, d.pay_amount, d.pay_currency, d.expires_at,
		       gl.id AS loc_id,
		       ST_Y(gl.location::geometry) AS lat,
		       ST_X(gl.location::geometry) AS lng,
		       gl.city, gl.district,
		       ST_Distance(gl.location, ST_SetSRID(ST_MakePoint($2,$1),4326)::geography) AS distance_m
		FROM gigs g
		JOIN gig_locations gl ON gl.gig_id = g.id
		LEFT JOIN gig_details d ON d.gig_id = g.id
		WHERE g.status = 'OPEN'
		  AND ST_DWithin(gl.location, ST_SetSRID(ST_MakePoint($2,$1),4326)::geography, $3)`
)

type GigRepository interface {
	Save(ctx context.Context, g *Gig, details *GigDetails, loc *GigLocation, categoryIDs []uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*GigFull, error)
	Update(ctx context.Context, id uuid.UUID, posterID uuid.UUID, in UpdateGigInput) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status GigStatus) error
	FindFeed(ctx context.Context, p FeedParams) ([]*GigFull, error)
}

type repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) GigRepository {
	return &repository{db: db}
}

func (r *repository) Save(ctx context.Context, g *Gig, details *GigDetails, loc *GigLocation, categoryIDs []uuid.UUID) error {
	db := dbtx.Extract(ctx, r.db)

	_, err := db.Exec(ctx, queryInsertGig,
		g.ID, g.PosterID, g.Title, g.DescriptionRaw, g.DescriptionClean, g.Status, g.CreatedAt,
	)

	if err != nil {
		return err
	}

	if details != nil {
		_, err = db.Exec(ctx, queryInsertGigDetails,
			g.ID, details.DurationType, details.StartDate, details.EndDate,
			details.PayAmount, details.PayCurrency, details.ExpiresAt,
		)
		if err != nil {
			return err
		}
	}

	if loc != nil {
		_, err = db.Exec(ctx, queryInsertGigLocation,
			loc.ID, loc.GigID, loc.Lng, loc.Lat, loc.City, loc.District,
		)
		if err != nil {
			return err
		}
	}

	for _, catID := range categoryIDs {
		if _, err = db.Exec(ctx, queryInsertGigCategory, g.ID, catID); err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*GigFull, error) {
	db := dbtx.Extract(ctx, r.db)

	row := db.QueryRow(ctx, queryFindByID, id)
	g, err := r.scanGig(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGigNotFound
		}
		return nil, err
	}

	full := &GigFull{Gig: *g}

	// details
	detRow := db.QueryRow(ctx, queryFindDetailsByGigID, id)
	det, err := r.scanDetails(detRow)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		full.Details = det
	}

	// location
	locRow := db.QueryRow(ctx, queryFindLocationByGigID, id)
	loc, err := r.scanLocation(locRow)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		full.Location = loc
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
		full.Categories = append(full.Categories, catID)
	}

	return full, nil
}

func (r *repository) Update(ctx context.Context, id uuid.UUID, posterID uuid.UUID, in UpdateGigInput) error {
	db := dbtx.Extract(ctx, r.db)

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

	_, err = db.Exec(ctx, queryUpdateGig, id, in.Title, in.DescriptionRaw, in.DescriptionClean)
	if err != nil {
		return err
	}

	// Upsert details if any optional field was provided.
	if in.DurationType != nil || in.StartDate != nil || in.EndDate != nil ||
		in.PayAmount != nil || in.PayCurrency != nil || in.ExpiresAt != nil {
		_, err = db.Exec(ctx, queryUpsertGigDetails,
			id, in.DurationType, in.StartDate, in.EndDate,
			in.PayAmount, in.PayCurrency, in.ExpiresAt,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status GigStatus) error {
	_, err := dbtx.Extract(ctx, r.db).Exec(ctx, queryUpdateGigStatus, id, status)
	return err
}

func (r *repository) FindFeed(ctx context.Context, p FeedParams) ([]*GigFull, error) {
	db := dbtx.Extract(ctx, r.db)

	q := queryFeedBase
	args := []any{p.Lat, p.Lng, p.RadiusMeters}
	argIdx := 4

	if p.DurationType != nil {
		q += ` AND d.duration_type = $` + itoa(argIdx)
		args = append(args, *p.DurationType)
		argIdx++
	}
	if p.MinPay != nil {
		q += ` AND d.pay_amount >= $` + itoa(argIdx)
		args = append(args, *p.MinPay)
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

	var feed []*GigFull
	for rows.Next() {
		g := &Gig{}
		det := &GigDetails{}
		loc := &GigLocation{}
		var distanceM float64

		err := rows.Scan(
			&g.ID, &g.PosterID, &g.Title, &g.DescriptionRaw, &g.DescriptionClean, &g.Status, &g.CreatedAt,
			&det.GigID, &det.DurationType, &det.StartDate, &det.EndDate, &det.PayAmount, &det.PayCurrency, &det.ExpiresAt,
			&loc.ID, &loc.Lat, &loc.Lng, &loc.City, &loc.District,
			&distanceM,
		)
		if err != nil {
			return nil, err
		}
		loc.GigID = g.ID

		full := &GigFull{Gig: *g, Location: loc}
		// Only attach details if the LEFT JOIN matched.
		if det.GigID != uuid.Nil {
			full.Details = det
		}
		feed = append(feed, full)
	}

	// Fetch categories per gig.
	for _, full := range feed {
		catRows, err := db.Query(ctx, queryFindCategoriesByGigID, full.Gig.ID)
		if err != nil {
			return nil, err
		}
		for catRows.Next() {
			var catID uuid.UUID
			if err := catRows.Scan(&catID); err != nil {
				catRows.Close()
				return nil, err
			}
			full.Categories = append(full.Categories, catID)
		}
		catRows.Close()
	}

	return feed, nil
}

func (r *repository) scanGig(row pgx.Row) (*Gig, error) {
	g := &Gig{}
	err := row.Scan(
		&g.ID, &g.PosterID, &g.Title, &g.DescriptionRaw, &g.DescriptionClean, &g.Status, &g.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (r *repository) scanDetails(row pgx.Row) (*GigDetails, error) {
	d := &GigDetails{}
	err := row.Scan(
		&d.GigID, &d.DurationType, &d.StartDate, &d.EndDate,
		&d.PayAmount, &d.PayCurrency, &d.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *repository) scanLocation(row pgx.Row) (*GigLocation, error) {
	l := &GigLocation{}
	err := row.Scan(&l.ID, &l.GigID, &l.Lat, &l.Lng, &l.City, &l.District)
	if err != nil {
		return nil, err
	}
	return l, nil
}

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

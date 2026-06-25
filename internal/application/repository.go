package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vsayfb/gig-platform-core-service/pkg/dbtx"
)

var (
	ErrNotFound       = errors.New("application: not found")
	ErrAlreadyApplied = errors.New("application: already applied to this gig")
	ErrNotApplicant   = errors.New("application: requester is not the applicant")
)

const (
	queryInsert = `
		INSERT INTO applications (id, gig_id, applicant_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	queryFindByID = `
		SELECT id, gig_id, applicant_id, status, created_at
		FROM applications WHERE id = $1`

	queryFindByGigID = `
		SELECT id, gig_id, applicant_id, status, created_at
		FROM applications WHERE gig_id = $1
		ORDER BY created_at ASC`

	queryFindByGigAndApplicant = `
		SELECT id, gig_id, applicant_id, status, created_at
		FROM applications WHERE gig_id = $1 AND applicant_id = $2`

	queryUpdateStatus = `
		UPDATE applications SET status = $2 WHERE id = $1`

	// Reject all PENDING applications for a gig except the hired one.
	queryRejectOthers = `
		UPDATE applications SET status = 'REJECTED'
		WHERE gig_id = $1 AND id != $2 AND status = 'PENDING'`

	// Rollback: set all HIRED and REJECTED applications back to PENDING for a gig.
	queryRollbackToOpen = `
		UPDATE applications SET status = 'PENDING'
		WHERE gig_id = $1 AND status IN ('HIRED', 'REJECTED')`
)

type ApplicationRepository interface {
	Save(ctx context.Context, a *Application) error
	FindByID(ctx context.Context, id uuid.UUID) (*Application, error)
	FindByGigID(ctx context.Context, gigID uuid.UUID) ([]*Application, error)
	FindByGigAndApplicant(ctx context.Context, gigID, applicantID uuid.UUID) (*Application, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error
	RejectOthers(ctx context.Context, gigID uuid.UUID, hiredID uuid.UUID) error
	RollbackToOpen(ctx context.Context, gigID uuid.UUID) error
}

type repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) ApplicationRepository {
	return &repository{db: db}
}

func (r *repository) Save(ctx context.Context, a *Application) error {
	db := dbtx.Extract(ctx, r.db)

	_, err := db.Exec(ctx, queryInsert,
		a.ID, a.GigID, a.ApplicantID, a.Status, a.CreatedAt,
	)

	return err
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*Application, error) {
	db := dbtx.Extract(ctx, r.db)

	row := db.QueryRow(ctx, queryFindByID, id)

	return r.scan(row)
}

func (r *repository) FindByGigID(ctx context.Context, gigID uuid.UUID) ([]*Application, error) {
	db := dbtx.Extract(ctx, r.db)

	rows, err := db.Query(ctx, queryFindByGigID, gigID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var apps []*Application

	for rows.Next() {
		a, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}

	return apps, nil
}

func (r *repository) FindByGigAndApplicant(ctx context.Context, gigID, applicantID uuid.UUID) (*Application, error) {
	db := dbtx.Extract(ctx, r.db)

	row := db.QueryRow(ctx, queryFindByGigAndApplicant, gigID, applicantID)

	return r.scan(row)
}

func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	db := dbtx.Extract(ctx, r.db)

	_, err := db.Exec(ctx, queryUpdateStatus, id, status)

	return err
}

func (r *repository) RejectOthers(ctx context.Context, gigID uuid.UUID, hiredID uuid.UUID) error {
	db := dbtx.Extract(ctx, r.db)

	_, err := db.Exec(ctx, queryRejectOthers, gigID, hiredID)

	return err
}

func (r *repository) RollbackToOpen(ctx context.Context, gigID uuid.UUID) error {
	db := dbtx.Extract(ctx, r.db)

	_, err := db.Exec(ctx, queryRollbackToOpen, gigID)

	return err
}

func (r *repository) scan(row pgx.Row) (*Application, error) {
	a := &Application{}

	err := row.Scan(&a.ID, &a.GigID, &a.ApplicantID, &a.Status, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return a, nil
}

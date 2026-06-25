package contract

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vsayfb/gig-platform-core-service/pkg/dbtx"
)

var ErrNotFound = errors.New("contract: not found")

const (
	queryInsert = `
		INSERT INTO contracts (id, application_id, gig_id, employer_id, employee_id, status, hired_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	queryFindByID = `
		SELECT id, application_id, gig_id, employer_id, employee_id, status, hired_at, completed_at
		FROM contracts WHERE id = $1`

	queryFindByApplicationID = `
		SELECT id, application_id, gig_id, employer_id, employee_id, status, hired_at, completed_at
		FROM contracts WHERE application_id = $1`

	queryUpdateStatus = `
		UPDATE contracts SET status = $2 WHERE id = $1`

	queryMarkCompleted = `
		UPDATE contracts SET status = 'COMPLETED', completed_at = NOW() WHERE id = $1`
)

type ContractRepository interface {
	Save(ctx context.Context, c *Contract) error
	FindByID(ctx context.Context, id uuid.UUID) (*Contract, error)
	FindByApplicationID(ctx context.Context, applicationID uuid.UUID) (*Contract, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error
	MarkCompleted(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *pgxpool.Pool
}

func NewConctractRepository(db *pgxpool.Pool) ContractRepository {
	return &repository{db: db}
}

func (r *repository) Save(ctx context.Context, c *Contract) error {
	db := dbtx.Extract(ctx, r.db)

	_, err := db.Exec(ctx, queryInsert,
		c.ID, c.ApplicationID, c.GigID, c.EmployerID, c.EmployeeID,
		c.Status, c.HiredAt,
	)

	return err
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*Contract, error) {
	row := dbtx.Extract(ctx, r.db).QueryRow(ctx, queryFindByApplicationID, id)

	return r.scan(row)
}

func (r *repository) FindByApplicationID(ctx context.Context, applicationID uuid.UUID) (*Contract, error) {
	row := dbtx.Extract(ctx, r.db).QueryRow(ctx, queryFindByApplicationID, applicationID)

	return r.scan(row)
}

func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	_, err := dbtx.Extract(ctx, r.db).Exec(ctx, queryUpdateStatus, status)

	return err
}

func (r *repository) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	_, err := dbtx.Extract(ctx, r.db).Exec(ctx, queryMarkCompleted, id)

	return err
}

func (r *repository) scan(row pgx.Row) (*Contract, error) {
	c := &Contract{}

	err := row.Scan(
		&c.ID, &c.ApplicationID, &c.GigID, &c.EmployerID, &c.EmployeeID,
		&c.Status, &c.HiredAt, &c.CompletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return c, nil
}

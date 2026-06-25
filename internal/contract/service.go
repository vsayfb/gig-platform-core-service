package contract

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vsayfb/gig-platform-core-service/internal/application"
	"github.com/vsayfb/gig-platform-core-service/internal/gig"
	"github.com/vsayfb/gig-platform-core-service/pkg/dbtx"
)

var (
	ErrNotParty          = errors.New("contract: caller is not a party to this contract")
	ErrNotEmployer       = errors.New("contract: caller is not the employer")
	ErrNotEmployee       = errors.New("contract: caller is not the employee")
	ErrInvalidTransition = errors.New("contract: invalid status transition")
	ErrAlreadyHired      = errors.New("contract: gig already has an active contract")
)

type ContractService interface {
	Get(ctx context.Context, id uuid.UUID, callerID uuid.UUID) (*Contract, error)
	Hire(ctx context.Context, applicationID uuid.UUID, employerID uuid.UUID) (*Contract, error)
	JobDone(ctx context.Context, contractID uuid.UUID, employeeID uuid.UUID) (*Contract, error)
	Approve(ctx context.Context, contractID uuid.UUID, employerID uuid.UUID) (*Contract, error)
	Dispute(ctx context.Context, contractID uuid.UUID, employerID uuid.UUID) (*Contract, error)
	Cancel(ctx context.Context, contractID uuid.UUID, callerID uuid.UUID) (*Contract, error)
}

type service struct {
	repo    ContractRepository
	appRepo application.ApplicationRepository
	gigRepo gig.GigRepository
	db      *pgxpool.Pool
}

func NewContractService(repo ContractRepository, appRepo application.ApplicationRepository, gigRepo gig.GigRepository, db *pgxpool.Pool) ContractService {
	return &service{repo: repo, appRepo: appRepo, gigRepo: gigRepo, db: db}
}

func (s *service) Get(ctx context.Context, id uuid.UUID, callerID uuid.UUID) (*Contract, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.EmployerID != callerID && c.EmployeeID != callerID {
		return nil, ErrNotParty
	}
	return c, nil
}

// Hire: employer hires an applicant.
// Transitions: Application → HIRED, others → REJECTED, Gig → IN_PROGRESS, Contract created (ACTIVE).
func (s *service) Hire(ctx context.Context, applicationID uuid.UUID, employerID uuid.UUID) (*Contract, error) {
	app, err := s.appRepo.FindByID(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	if app.Status != application.StatusPending {
		return nil, ErrInvalidTransition
	}

	gigDetail, err := s.gigRepo.FindByID(ctx, app.GigID)

	if err != nil {
		return nil, err
	}

	if gigDetail.Gig.PosterID != employerID {
		return nil, ErrNotEmployer
	}

	if gigDetail.Gig.Status != gig.StatusOpen {
		return nil, ErrAlreadyHired
	}

	now := time.Now().UTC()

	c := &Contract{
		ID:            uuid.New(),
		ApplicationID: applicationID,
		GigID:         app.GigID,
		EmployerID:    employerID,
		EmployeeID:    app.ApplicantID,
		Status:        StatusActive,
		HiredAt:       now,
	}

	err = dbtx.RunInTx(ctx, s.db, func(ctx context.Context) error {

		if err := s.repo.Save(ctx, c); err != nil {
			return err
		}
		if err := s.appRepo.UpdateStatus(ctx, applicationID, application.StatusHired); err != nil {
			return err
		}
		if err := s.appRepo.RejectOthers(ctx, app.GigID, applicationID); err != nil {
			return err
		}
		if err := s.gigRepo.UpdateStatus(ctx, app.GigID, gig.StatusInProgress); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		slog.ErrorContext(ctx, "contract.Hire: transaction failed", "err", err)

		return nil, err
	}

	return c, nil
}

// JobDone: employee marks work complete, awaiting employer approval.
func (s *service) JobDone(ctx context.Context, contractID uuid.UUID, employeeID uuid.UUID) (*Contract, error) {
	c, err := s.repo.FindByID(ctx, contractID)

	if err != nil {
		return nil, err
	}

	if c.EmployeeID != employeeID {
		return nil, ErrNotEmployee
	}

	if c.Status != StatusActive {
		return nil, ErrInvalidTransition
	}

	if err := s.repo.UpdateStatus(ctx, contractID, StatusAwaitingApproval); err != nil {
		return nil, err
	}

	c.Status = StatusAwaitingApproval

	return c, nil
}

// Approve: employer confirms completion.
// Transitions: Contract → COMPLETED, Gig → COMPLETED, Application → COMPLETED.
func (s *service) Approve(ctx context.Context, contractID uuid.UUID, employerID uuid.UUID) (*Contract, error) {
	c, err := s.repo.FindByID(ctx, contractID)

	if err != nil {
		return nil, err
	}

	if c.EmployerID != employerID {
		return nil, ErrNotEmployer
	}

	if c.Status != StatusAwaitingApproval {
		return nil, ErrInvalidTransition
	}

	err = dbtx.RunInTx(ctx, s.db, func(ctx context.Context) error {

		if err := s.repo.MarkCompleted(ctx, contractID); err != nil {
			return err
		}
		if err := s.appRepo.UpdateStatus(ctx, c.ApplicationID, application.StatusCompleted); err != nil {
			return err
		}
		if err := s.gigRepo.UpdateStatus(ctx, c.GigID, gig.StatusCompleted); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		slog.ErrorContext(ctx, "contract.Approve: transaction failed", "err", err)
		return nil, err
	}

	return s.repo.FindByID(ctx, contractID)
}

// Dispute: employer disputes instead of approving.
func (s *service) Dispute(ctx context.Context, contractID uuid.UUID, employerID uuid.UUID) (*Contract, error) {
	c, err := s.repo.FindByID(ctx, contractID)
	if err != nil {
		return nil, err
	}
	if c.EmployerID != employerID {
		return nil, ErrNotEmployer
	}
	if c.Status != StatusAwaitingApproval {
		return nil, ErrInvalidTransition
	}

	if err := s.repo.UpdateStatus(ctx, contractID, StatusDisputed); err != nil {
		return nil, err
	}
	c.Status = StatusDisputed
	return c, nil
}

// Cancel: either party can cancel. Triggers full rollback per spec.
// Transitions: Contract → CANCELLED, Gig → OPEN, all HIRED+REJECTED applications → PENDING.
func (s *service) Cancel(ctx context.Context, contractID uuid.UUID, callerID uuid.UUID) (*Contract, error) {
	c, err := s.repo.FindByID(ctx, contractID)

	if err != nil {
		return nil, err
	}

	if c.EmployerID != callerID && c.EmployeeID != callerID {
		return nil, ErrNotParty
	}

	if c.Status != StatusActive && c.Status != StatusAwaitingApproval {
		return nil, ErrInvalidTransition
	}

	err = dbtx.RunInTx(ctx, s.db, func(ctx context.Context) error {

		if err := s.repo.UpdateStatus(ctx, contractID, StatusCancelled); err != nil {
			return err
		}
		if err := s.appRepo.RollbackToOpen(ctx, c.GigID); err != nil {
			return err
		}
		if err := s.gigRepo.UpdateStatus(ctx, c.GigID, gig.StatusOpen); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		slog.ErrorContext(ctx, "contract.Cancel: transaction failed", "err", err)
		return nil, err
	}

	return s.repo.FindByID(ctx, contractID)
}

package contract

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive           Status = "ACTIVE"
	StatusAwaitingApproval Status = "AWAITING_APPROVAL"
	StatusCompleted        Status = "COMPLETED"
	StatusDisputed         Status = "DISPUTED"
	StatusCancelled        Status = "CANCELLED"
)

type Contract struct {
	ID            uuid.UUID  `json:"id"`
	ApplicationID uuid.UUID  `json:"application_id"`
	GigID         uuid.UUID  `json:"gig_id"`
	EmployerID    uuid.UUID  `json:"employer_id"`
	EmployeeID    uuid.UUID  `json:"employee_id"`
	Status        Status     `json:"status"`
	HiredAt       time.Time  `json:"hired_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

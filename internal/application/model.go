package application

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending   Status = "PENDING"
	StatusHired     Status = "HIRED"
	StatusRejected  Status = "REJECTED"
	StatusWithdrawn Status = "WITHDRAWN"
	StatusCompleted Status = "COMPLETED"
)

type Application struct {
	ID          uuid.UUID `json:"id"`
	GigID       uuid.UUID `json:"gig_id"`
	ApplicantID uuid.UUID `json:"applicant_id"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

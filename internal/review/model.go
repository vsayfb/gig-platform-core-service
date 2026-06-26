package review

import (
	"time"

	"github.com/google/uuid"
)

type RoleContext string

const (
	RoleAsEmployer RoleContext = "AS_EMPLOYER"
	RoleAsEmployee RoleContext = "AS_EMPLOYEE"
)

type Review struct {
	ID          uuid.UUID   `json:"id"`
	ContractID  uuid.UUID   `json:"contract_id"`
	ReviewerID  uuid.UUID   `json:"reviewer_id"`
	RevieweeID  uuid.UUID   `json:"reviewee_id"`
	Rating      int16       `json:"rating"`
	Comment     string      `json:"comment"`
	RoleContext RoleContext `json:"role_context"`
	CreatedAt   time.Time   `json:"created_at"`
}

type CreateReviewInput struct {
	Rating  int16  `json:"rating"`
	Comment string `json:"comment"`
}

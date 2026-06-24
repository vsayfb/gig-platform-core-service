package category

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusPending  Status = "PENDING"
	StatusRejected Status = "REJECTED"
)

type Category struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Status    Status
	CreatedAt time.Time
}

func NewCategory(name, slug string) *Category {
	return &Category{
		Name: name,
		Slug: slug,
	}
}

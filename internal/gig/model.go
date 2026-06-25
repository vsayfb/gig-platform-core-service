package gig

import (
	"time"

	"github.com/google/uuid"
)

type DurationType string

const (
	DurationDaily   DurationType = "DAILY"
	DurationWeekly  DurationType = "WEEKLY"
	DurationMonthly DurationType = "MONTHLY"
)

type GigStatus string

const (
	StatusOpen       GigStatus = "OPEN"
	StatusInProgress GigStatus = "IN_PROGRESS"
	StatusCompleted  GigStatus = "COMPLETED"
	StatusCancelled  GigStatus = "CANCELLED"
)

type Gig struct {
	ID               uuid.UUID    `json:"id"`
	PosterID         uuid.UUID    `json:"poster_id"`
	Title            string       `json:"title"`
	DescriptionRaw   string       `json:"description_raw"`
	DescriptionClean string       `json:"description_clean"`
	DurationType     DurationType `json:"duration_type"`
	StartDate        time.Time    `json:"start_date"`
	EndDate          *time.Time   `json:"end_date,omitempty"`
	Slots            int          `json:"slots"`
	Status           GigStatus    `json:"status"`
	CreatedAt        time.Time    `json:"created_at"`
	ExpiresAt        *time.Time   `json:"expires_at,omitempty"`
}

type GigLocation struct {
	ID       uuid.UUID `json:"id"`
	GigID    uuid.UUID `json:"gig_id"`
	Lat      float64   `json:"lat"`
	Lng      float64   `json:"lng"`
	City     string    `json:"city"`
	District string    `json:"district"`
}

type GigCategory struct {
	GigID      uuid.UUID `json:"gig_id"`
	CategoryID uuid.UUID `json:"category_id"`
}

type GigDetail struct {
	Gig
	Location   *GigLocation `json:"location,omitempty"`
	Categories []uuid.UUID  `json:"categories"`
}

type FeedParams struct {
	Lat          float64
	Lng          float64
	RadiusMeters float64
	DurationType *DurationType
	CategoryID   *uuid.UUID
	Cursor       *time.Time // created_at of last seen gig (keyset pagination)
	Limit        int
}

type CreateGigInput struct {
	Title            string       `json:"title"`
	DescriptionRaw   string       `json:"description_raw"`
	DescriptionClean string       `json:"description_clean"`
	DurationType     DurationType `json:"duration_type"`
	StartDate        time.Time    `json:"start_date"`
	EndDate          *time.Time   `json:"end_date,omitempty"`
	Slots            int          `json:"slots"`
	ExpiresAt        *time.Time   `json:"expires_at,omitempty"`
	Lat              float64      `json:"lat"`
	Lng              float64      `json:"lng"`
	City             string       `json:"city"`
	District         string       `json:"district"`
	CategoryIDs      []uuid.UUID  `json:"category_ids"`
}

type UpdateGigInput struct {
	Title            *string       `json:"title,omitempty"`
	DescriptionRaw   *string       `json:"description_raw,omitempty"`
	DescriptionClean *string       `json:"description_clean,omitempty"`
	DurationType     *DurationType `json:"duration_type,omitempty"`
	StartDate        *time.Time    `json:"start_date,omitempty"`
	EndDate          *time.Time    `json:"end_date,omitempty"`
	Slots            *int          `json:"slots,omitempty"`
	ExpiresAt        *time.Time    `json:"expires_at,omitempty"`
}

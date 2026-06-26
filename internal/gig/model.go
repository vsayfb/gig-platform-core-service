package gig

import (
	"time"

	"github.com/google/uuid"
)

const RADIUS_METERS = 5000

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

// Gig is the core record — only mandatory fields.
type Gig struct {
	ID               uuid.UUID `json:"id"`
	PosterID         uuid.UUID `json:"poster_id"`
	Title            string    `json:"title"`
	DescriptionRaw   string    `json:"description_raw"`
	DescriptionClean string    `json:"description_clean"`
	Status           GigStatus `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}

// GigDetails holds optional poster-provided fields, 1-to-1 with Gig.
type GigDetails struct {
	GigID        uuid.UUID     `json:"gig_id"`
	DurationType *DurationType `json:"duration_type,omitempty"`
	StartDate    *time.Time    `json:"start_date,omitempty"`
	EndDate      *time.Time    `json:"end_date,omitempty"`
	PayAmount    *float64      `json:"pay_amount,omitempty"`
	PayCurrency  *string       `json:"pay_currency,omitempty"`
	ExpiresAt    *time.Time    `json:"expires_at,omitempty"`
}

type GigLocation struct {
	ID       uuid.UUID `json:"id"`
	GigID    uuid.UUID `json:"gig_id"`
	Lat      float64   `json:"lat"`
	Lng      float64   `json:"lng"`
	City     string    `json:"city"`
	District string    `json:"district"`
}

// GigFull is the complete view returned on GET /gigs/:id and feed items.
type GigFull struct {
	Gig
	Details    *GigDetails  `json:"details,omitempty"`
	Location   *GigLocation `json:"location,omitempty"`
	Categories []uuid.UUID  `json:"categories"`
}

// FeedParams holds query params for GET /gigs.
type FeedParams struct {
	Lat          float64
	Lng          float64
	RadiusMeters float64
	DurationType *DurationType
	MinPay       *float64
	CategoryID   *uuid.UUID
	Cursor       *time.Time
	Limit        int
}

// CreateGigInput is the payload for POST /gigs.
type CreateGigInput struct {
	Title            string `json:"title"`
	DescriptionRaw   string `json:"description_raw"`
	DescriptionClean string `json:"description_clean"`

	// Optional details
	DurationType *DurationType `json:"duration_type,omitempty"`
	StartDate    *time.Time    `json:"start_date,omitempty"`
	EndDate      *time.Time    `json:"end_date,omitempty"`
	PayAmount    *float64      `json:"pay_amount,omitempty"`
	PayCurrency  *string       `json:"pay_currency,omitempty"`
	ExpiresAt    *time.Time    `json:"expires_at,omitempty"`

	// Location
	Lat      *float64 `json:"lat"`
	Lng      *float64 `json:"lng"`
	City     *string  `json:"city"`
	District *string  `json:"district"`

	// Categories (optional)
	CategoryIDs []uuid.UUID `json:"category_ids,omitempty"`
}

// UpdateGigInput is the payload for PUT /gigs/:id.
type UpdateGigInput struct {
	Title            *string       `json:"title,omitempty"`
	DescriptionRaw   *string       `json:"description_raw,omitempty"`
	DescriptionClean *string       `json:"description_clean,omitempty"`
	DurationType     *DurationType `json:"duration_type,omitempty"`
	StartDate        *time.Time    `json:"start_date,omitempty"`
	EndDate          *time.Time    `json:"end_date,omitempty"`
	PayAmount        *float64      `json:"pay_amount,omitempty"`
	PayCurrency      *string       `json:"pay_currency,omitempty"`
	ExpiresAt        *time.Time    `json:"expires_at,omitempty"`
}

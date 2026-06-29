package squs

import (
	"context"

	"github.com/google/uuid"
)

type GigLocation struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type GigCreatedEvent struct {
	GigID       uuid.UUID   `json:"gig_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Location    GigLocation `json:"location"`
}

type EventPublisher interface {
	Publish(ctx context.Context, event GigCreatedEvent) error
}

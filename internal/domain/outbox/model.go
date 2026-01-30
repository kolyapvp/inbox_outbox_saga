package outbox

import (
	"context"
	"time"
)

type Event struct {
	ID            string    `json:"id"`
	EventType     string    `json:"event_type"`
	Payload       []byte    `json:"payload"`
	Status        string    `json:"status"`
	CorrelationID string    `json:"correlation_id"`
	CausationID   string    `json:"causation_id"`
	Producer      string    `json:"producer"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Repository interface {
	Create(ctx context.Context, event *Event) error
	FetchBatch(ctx context.Context, limit int) ([]*Event, error)
	MarkProcessed(ctx context.Context, ids []string) error
}

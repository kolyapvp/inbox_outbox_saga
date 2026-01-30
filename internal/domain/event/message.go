package event

import (
	"encoding/json"
	"time"
)

// Message is the envelope published to Kafka.
// Payload is kept as raw JSON produced by the originating service.
type Message struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	CorrelationID string          `json:"correlation_id"`
	CausationID   string          `json:"causation_id,omitempty"`
	Producer      string          `json:"producer"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Payload       json.RawMessage `json:"payload"`
}

package inbox

import "time"

// Event is a consumer-side record used for deduplication (Inbox pattern).
// Each consumer stores processed event IDs with metadata.
type Event struct {
	Consumer      string    `json:"consumer"`
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	CorrelationID string    `json:"correlation_id"`
	ProcessedAt   time.Time `json:"processed_at"`
}

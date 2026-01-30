package postgres

import (
	"context"
	"fmt"

	"project/internal/domain/inbox"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InboxRepository struct {
	pool *pgxpool.Pool
}

func NewInboxRepository(pool *pgxpool.Pool) *InboxRepository {
	return &InboxRepository{pool: pool}
}

// SaveIfNotExists returns true if the event was saved (is new), false if it already existed.
func (r *InboxRepository) SaveIfNotExists(ctx context.Context, tx pgx.Tx, consumer string, eventID string, eventType string, correlationID string) (bool, error) {
	const query = `
		INSERT INTO inbox_events (consumer, event_id, event_type, correlation_id, processed_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (consumer, event_id) DO NOTHING
	`

	tag, err := tx.Exec(ctx, query, consumer, eventID, eventType, nullIfEmptyText(correlationID))
	if err != nil {
		return false, fmt.Errorf("insert inbox event: %w", err)
	}

	return tag.RowsAffected() > 0, nil
}

func (r *InboxRepository) ListByCorrelationID(ctx context.Context, correlationID string) ([]*inbox.Event, error) {
	const query = `
		SELECT consumer, event_id, event_type, correlation_id, processed_at
		FROM inbox_events
		WHERE correlation_id = $1
		ORDER BY processed_at ASC
	`

	rows, err := r.pool.Query(ctx, query, nullIfEmptyText(correlationID))
	if err != nil {
		return nil, fmt.Errorf("query inbox events: %w", err)
	}
	defer rows.Close()

	var events []*inbox.Event
	for rows.Next() {
		e := &inbox.Event{}
		if err := rows.Scan(&e.Consumer, &e.EventID, &e.EventType, &e.CorrelationID, &e.ProcessedAt); err != nil {
			return nil, fmt.Errorf("scan inbox event: %w", err)
		}
		events = append(events, e)
	}

	return events, nil
}

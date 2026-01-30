package postgres

import (
	"context"
	"fmt"
	"project/internal/domain/outbox"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

func (r *OutboxRepository) Create(ctx context.Context, e *outbox.Event) error {
	const sql = `
		INSERT INTO outbox (id, event_type, payload, status, correlation_id, causation_id, producer, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`

	var executor interface {
		Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	} = r.pool

	if tx := GetTx(ctx); tx != nil {
		executor = tx
	}

	_, err := executor.Exec(ctx, sql,
		e.ID, e.EventType, e.Payload, e.Status, nullIfEmpty(e.CorrelationID), nullIfEmpty(e.CausationID), nullIfEmptyDefault(e.Producer, "unknown"), e.CreatedAt)

	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}

	return nil
}

func (r *OutboxRepository) FetchBatch(ctx context.Context, limit int) ([]*outbox.Event, error) {
	const sql = `
		WITH claimed_events AS (
			SELECT id
			FROM outbox
			WHERE status = 'new'
			ORDER BY created_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE outbox
		SET status = 'processing', updated_at = NOW()
		WHERE id IN (SELECT id FROM claimed_events)
		RETURNING
			id,
			event_type,
			payload,
			status,
			COALESCE(correlation_id::text, ''),
			COALESCE(causation_id::text, ''),
			COALESCE(producer, 'unknown'),
			created_at,
			updated_at
	`

	rows, err := r.pool.Query(ctx, sql, limit)
	if err != nil {
		return nil, fmt.Errorf("query outbox: %w", err)
	}
	defer rows.Close()

	var events []*outbox.Event
	for rows.Next() {
		e := &outbox.Event{}
		if err := rows.Scan(&e.ID, &e.EventType, &e.Payload, &e.Status, &e.CorrelationID, &e.CausationID, &e.Producer, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}

	return events, nil
}

func (r *OutboxRepository) MarkProcessed(ctx context.Context, ids []string) error {
	const sql = `
		UPDATE outbox
		SET status = 'processed', updated_at = NOW()
		WHERE id = ANY($1)
	`
	_, err := r.pool.Exec(ctx, sql, ids)
	if err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}
	return nil
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, ids []string) error {
	const sql = `
		UPDATE outbox
		SET status = 'new', updated_at = NOW()
		WHERE id = ANY($1)
	`
	_, err := r.pool.Exec(ctx, sql, ids)
	if err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}
	return nil
}

func (r *OutboxRepository) ListByCorrelationID(ctx context.Context, correlationID string) ([]*outbox.Event, error) {
	const sql = `
		SELECT
			id,
			event_type,
			payload,
			status,
			COALESCE(correlation_id::text, ''),
			COALESCE(causation_id::text, ''),
			COALESCE(producer, 'unknown'),
			created_at,
			updated_at
		FROM outbox
		WHERE correlation_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, sql, nullIfEmpty(correlationID))
	if err != nil {
		return nil, fmt.Errorf("query outbox by correlation_id: %w", err)
	}
	defer rows.Close()

	var events []*outbox.Event
	for rows.Next() {
		e := &outbox.Event{}
		if err := rows.Scan(&e.ID, &e.EventType, &e.Payload, &e.Status, &e.CorrelationID, &e.CausationID, &e.Producer, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		events = append(events, e)
	}

	return events, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullIfEmptyDefault(s string, def string) any {
	if s == "" {
		return def
	}
	return s
}

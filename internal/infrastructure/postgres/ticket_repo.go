package postgres

import (
	"context"
	"fmt"

	"project/internal/domain/ticket"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TicketRepository struct {
	pool *pgxpool.Pool
}

func NewTicketRepository(pool *pgxpool.Pool) *TicketRepository {
	return &TicketRepository{pool: pool}
}

func (r *TicketRepository) Create(ctx context.Context, t *ticket.Ticket) error {
	const sql = `
		INSERT INTO tickets (
			id, order_id,
			from_city, to_city, travel_date, travel_time, airline,
			status, created_at, updated_at
		)
		VALUES (
			$1, $2,
			$3, $4, NULLIF($5, '')::date, $6, $7,
			$8, $9, $10
		)
		ON CONFLICT (order_id) DO NOTHING
	`

	var executor interface {
		Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	} = r.pool

	if tx := GetTx(ctx); tx != nil {
		executor = tx
	}

	_, err := executor.Exec(
		ctx,
		sql,
		t.ID, t.OrderID,
		nullIfEmptyText(t.FromCity), nullIfEmptyText(t.ToCity), t.TravelDate, nullIfEmptyText(t.TravelTime), nullIfEmptyText(t.Airline),
		t.Status, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert ticket: %w", err)
	}

	return nil
}

func (r *TicketRepository) GetByOrderID(ctx context.Context, orderID string) (*ticket.Ticket, error) {
	const sql = `
		SELECT
			id, order_id,
			COALESCE(from_city, ''),
			COALESCE(to_city, ''),
			COALESCE(to_char(travel_date, 'YYYY-MM-DD'), ''),
			COALESCE(travel_time, ''),
			COALESCE(airline, ''),
			status, created_at, updated_at
		FROM tickets
		WHERE order_id = $1
	`

	var t ticket.Ticket
	err := r.pool.QueryRow(ctx, sql, orderID).Scan(
		&t.ID, &t.OrderID,
		&t.FromCity, &t.ToCity, &t.TravelDate, &t.TravelTime, &t.Airline,
		&t.Status, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get ticket by order_id: %w", err)
	}
	return &t, nil
}

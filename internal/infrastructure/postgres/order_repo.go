package postgres

import (
	"context"
	"fmt"
	"project/internal/domain/order"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

func (r *OrderRepository) Create(ctx context.Context, o *order.Order) error {
	const sql = `
		INSERT INTO orders (
			id, user_id, status, total_amount,
			from_city, to_city, travel_date, travel_time, airline,
			created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4,
			$5, $6, NULLIF($7, '')::date, $8, $9,
			$10, $11
		)
	`

	// Check for transaction in context
	var executor interface {
		Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	} = r.pool

	if tx := GetTx(ctx); tx != nil {
		executor = tx
	}

	_, err := executor.Exec(ctx, sql,
		o.ID, o.UserID, o.Status, o.TotalAmount,
		nullIfEmptyText(o.FromCity), nullIfEmptyText(o.ToCity), o.TravelDate, nullIfEmptyText(o.TravelTime), nullIfEmptyText(o.Airline),
		o.CreatedAt, o.UpdatedAt)

	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	return nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	const sql = `
		UPDATE orders
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`

	var executor interface {
		Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	} = r.pool

	if tx := GetTx(ctx); tx != nil {
		executor = tx
	}

	cmdTag, err := executor.Exec(ctx, sql, id, status)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("order not found")
	}

	return nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id string) (*order.Order, error) {
	const sql = `
		SELECT
			id, user_id, status, total_amount,
			COALESCE(from_city, ''),
			COALESCE(to_city, ''),
			COALESCE(to_char(travel_date, 'YYYY-MM-DD'), ''),
			COALESCE(travel_time, ''),
			COALESCE(airline, ''),
			created_at, updated_at
		FROM orders
		WHERE id = $1
	`

	var o order.Order
	err := r.pool.QueryRow(ctx, sql, id).Scan(
		&o.ID, &o.UserID, &o.Status, &o.TotalAmount,
		&o.FromCity, &o.ToCity, &o.TravelDate, &o.TravelTime, &o.Airline,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get order by id: %w", err)
	}

	return &o, nil
}

func nullIfEmptyText(s string) any {
	if s == "" {
		return nil
	}
	return s
}

package postgres

import (
	"context"
	"fmt"

	"project/internal/domain/payment"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentRepository struct {
	pool *pgxpool.Pool
}

func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{pool: pool}
}

func (r *PaymentRepository) Create(ctx context.Context, p *payment.Payment) error {
	const sql = `
		INSERT INTO payments (id, order_id, status, amount, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (order_id) DO NOTHING
	`

	var executor interface {
		Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	} = r.pool

	if tx := GetTx(ctx); tx != nil {
		executor = tx
	}

	_, err := executor.Exec(ctx, sql, p.ID, p.OrderID, p.Status, p.Amount, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}

	return nil
}

func (r *PaymentRepository) GetByOrderID(ctx context.Context, orderID string) (*payment.Payment, error) {
	const sql = `
		SELECT id, order_id, status, amount, created_at, updated_at
		FROM payments
		WHERE order_id = $1
	`

	var p payment.Payment
	err := r.pool.QueryRow(ctx, sql, orderID).Scan(&p.ID, &p.OrderID, &p.Status, &p.Amount, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get payment by order_id: %w", err)
	}
	return &p, nil
}

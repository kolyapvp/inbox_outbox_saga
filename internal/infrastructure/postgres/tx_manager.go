package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Transactor interface {
	WithinTransaction(ctx context.Context, tFunc func(ctx context.Context) error) error
}

type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// WithinTransaction executes a function within a transaction.
// It injects the tx into the context.
func (tm *TxManager) WithinTransaction(ctx context.Context, tFunc func(ctx context.Context) error) error {
	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Defer rollback in case of panic or error
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	// Inject transaction into context
	// In a real project, use a custom key type to avoid collisions
	ctxWithTx := context.WithValue(ctx, "tx", tx)

	err = tFunc(ctxWithTx)
	return err
}

// GetTx retrieves the transaction from context, or nil if not present.
// This allows repositories to support both transactional and non-transactional modes.
func GetTx(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value("tx").(pgx.Tx); ok {
		return tx
	}
	return nil
}

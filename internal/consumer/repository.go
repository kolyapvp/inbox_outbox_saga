package consumer

import (
	"context"
	"fmt"

	"project/internal/infrastructure/postgres"

	"github.com/jackc/pgx/v5"
)

type ProcessedEventRepository struct {
	txManager *postgres.TxManager
}

func NewProcessedEventRepository(txManager *postgres.TxManager) *ProcessedEventRepository {
	return &ProcessedEventRepository{
		txManager: txManager,
	}
}

// SaveIfNotExists returns true if the event was saved (is new), false if it already existed.
func (r *ProcessedEventRepository) SaveIfNotExists(ctx context.Context, tx pgx.Tx, eventID string) (bool, error) {
	query := `
		INSERT INTO processed_events (event_id, processed_at)
		VALUES ($1, NOW())
		ON CONFLICT (event_id) DO NOTHING
	`
	tag, err := tx.Exec(ctx, query, eventID)
	if err != nil {
		return false, fmt.Errorf("failed to insert processed event: %w", err)
	}

	return tag.RowsAffected() > 0, nil
}

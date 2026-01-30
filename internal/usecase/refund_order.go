package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"project/internal/domain/outbox"
	"project/internal/infrastructure/postgres"

	"github.com/google/uuid"
)

type RefundOrder struct {
	txManager  postgres.Transactor
	orderRepo  *postgres.OrderRepository
	outboxRepo *postgres.OutboxRepository
}

func NewRefundOrder(
	txManager postgres.Transactor,
	orderRepo *postgres.OrderRepository,
	outboxRepo *postgres.OutboxRepository,
) *RefundOrder {
	return &RefundOrder{
		txManager:  txManager,
		orderRepo:  orderRepo,
		outboxRepo: outboxRepo,
	}
}

type RefundOrderParams struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

type RefundEvent struct {
	OrderID   string    `json:"order_id"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

func (uc *RefundOrder) Execute(ctx context.Context, params RefundOrderParams) error {
	// Prepare outbox event
	eventPayload := RefundEvent{
		OrderID:   params.OrderID,
		Reason:    params.Reason,
		Timestamp: time.Now(),
	}

	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("marshal refund event: %w", err)
	}

	outboxEvent := &outbox.Event{
		ID:            uuid.New().String(),
		EventType:     "RefundInitiated",
		Payload:       payload,
		Status:        "new",
		CorrelationID: params.OrderID,
		CausationID:   "",
		Producer:      "order-service",
		CreatedAt:     time.Now(),
	}

	// Execute in transaction
	err = uc.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		// 1. Update Order Status
		if err := uc.orderRepo.UpdateStatus(txCtx, params.OrderID, "REFUND_PENDING"); err != nil {
			return err
		}

		// 2. Save Outbox Event
		if err := uc.outboxRepo.Create(txCtx, outboxEvent); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

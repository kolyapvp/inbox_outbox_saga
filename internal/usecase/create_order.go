package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"project/internal/domain/order"
	"project/internal/domain/outbox"
	"project/internal/infrastructure/postgres"

	"github.com/google/uuid"
)

type CreateOrder struct {
	txManager  postgres.Transactor
	orderRepo  *postgres.OrderRepository
	outboxRepo *postgres.OutboxRepository
}

func NewCreateOrder(
	txManager postgres.Transactor,
	orderRepo *postgres.OrderRepository,
	outboxRepo *postgres.OutboxRepository,
) *CreateOrder {
	return &CreateOrder{
		txManager:  txManager,
		orderRepo:  orderRepo,
		outboxRepo: outboxRepo,
	}
}

type CreateOrderParams struct {
	UserID  string  `json:"user_id"`
	Amount  float64 `json:"amount"`
	From    string  `json:"from"`
	To      string  `json:"to"`
	Date    string  `json:"date"`
	Time    string  `json:"time"`
	Airline string  `json:"airline"`
}

func (uc *CreateOrder) Execute(ctx context.Context, params CreateOrderParams) (string, error) {
	newOrder := &order.Order{
		ID:          uuid.New().String(),
		UserID:      params.UserID,
		Status:      "CREATED",
		TotalAmount: params.Amount,
		FromCity:    params.From,
		ToCity:      params.To,
		TravelDate:  params.Date,
		TravelTime:  params.Time,
		Airline:     params.Airline,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Prepare outbox event
	payload, err := json.Marshal(newOrder)
	if err != nil {
		return "", fmt.Errorf("marshal order: %w", err)
	}

	outboxEvent := &outbox.Event{
		ID:            uuid.New().String(),
		EventType:     "OrderCreated",
		Payload:       payload,
		Status:        "new",
		CorrelationID: newOrder.ID,
		CausationID:   "",
		Producer:      "order-service",
		CreatedAt:     time.Now(),
	}

	// Execute in transaction
	err = uc.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.orderRepo.Create(txCtx, newOrder); err != nil {
			return err
		}

		if err := uc.outboxRepo.Create(txCtx, outboxEvent); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("transaction failed: %w", err)
	}

	return newOrder.ID, nil
}

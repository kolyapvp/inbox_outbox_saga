package usecase

import (
	"context"
	"fmt"

	"project/internal/domain/inbox"
	"project/internal/domain/outbox"
	"project/internal/domain/payment"
	"project/internal/domain/ticket"
	"project/internal/infrastructure/postgres"
)

type WorkflowDTO struct {
	Order   *OrderDTO        `json:"order"`
	Outbox  []*outbox.Event  `json:"outbox"`
	Inbox   []*inbox.Event   `json:"inbox"`
	Payment *payment.Payment `json:"payment,omitempty"`
	Ticket  *ticket.Ticket   `json:"ticket,omitempty"`
}

type GetWorkflow struct {
	orderRepo   *postgres.OrderRepository
	outboxRepo  *postgres.OutboxRepository
	inboxRepo   *postgres.InboxRepository
	paymentRepo *postgres.PaymentRepository
	ticketRepo  *postgres.TicketRepository
}

func NewGetWorkflow(
	orderRepo *postgres.OrderRepository,
	outboxRepo *postgres.OutboxRepository,
	inboxRepo *postgres.InboxRepository,
	paymentRepo *postgres.PaymentRepository,
	ticketRepo *postgres.TicketRepository,
) *GetWorkflow {
	return &GetWorkflow{
		orderRepo:   orderRepo,
		outboxRepo:  outboxRepo,
		inboxRepo:   inboxRepo,
		paymentRepo: paymentRepo,
		ticketRepo:  ticketRepo,
	}
}

func (uc *GetWorkflow) Execute(ctx context.Context, orderID string) (*WorkflowDTO, error) {
	dbOrder, err := uc.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	order := &OrderDTO{
		ID:          dbOrder.ID,
		UserID:      dbOrder.UserID,
		TotalAmount: dbOrder.TotalAmount,
		Status:      dbOrder.Status,
		FromCity:    dbOrder.FromCity,
		ToCity:      dbOrder.ToCity,
		TravelDate:  dbOrder.TravelDate,
		TravelTime:  dbOrder.TravelTime,
		Airline:     dbOrder.Airline,
		CreatedAt:   dbOrder.CreatedAt,
	}

	outboxEvents, err := uc.outboxRepo.ListByCorrelationID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get outbox events: %w", err)
	}

	inboxEvents, err := uc.inboxRepo.ListByCorrelationID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get inbox events: %w", err)
	}

	p, err := uc.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get payment: %w", err)
	}

	t, err := uc.ticketRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get ticket: %w", err)
	}

	return &WorkflowDTO{
		Order:   order,
		Outbox:  outboxEvents,
		Inbox:   inboxEvents,
		Payment: p,
		Ticket:  t,
	}, nil
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"project/internal/application/factories/infrastructure"
	"project/internal/config"
	domainEvent "project/internal/domain/event"
	"project/internal/domain/outbox"
	"project/internal/domain/ticket"
	"project/internal/infrastructure/kafka"
	"project/internal/infrastructure/postgres"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ticketsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ticket_service_events_processed_total",
		Help: "The total number of processed events by ticket service",
	})
)

type paymentAuthorizedPayload struct {
	OrderID    string  `json:"order_id"`
	PaymentID  string  `json:"payment_id"`
	Amount     float64 `json:"amount"`
	FromCity   string  `json:"from_city"`
	ToCity     string  `json:"to_city"`
	TravelDate string  `json:"travel_date"`
	TravelTime string  `json:"travel_time"`
	Airline    string  `json:"airline"`
}

type ticketIssuedPayload struct {
	OrderID  string `json:"order_id"`
	TicketID string `json:"ticket_id"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.New()
	if err != nil {
		logger.Error("Failed to load config, using defaults", "error", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		logger.Info("Ticket metrics listening on :9095")
		http.ListenAndServe(":9095", mux)
	}()

	infraFactory := infrastructure.NewFactory(cfg)
	defer infraFactory.Close()

	pgPool, err := infraFactory.Postgres(ctx)
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}

	inboxRepo := postgres.NewInboxRepository(pgPool)
	outboxRepo := postgres.NewOutboxRepository(pgPool)
	ticketRepo := postgres.NewTicketRepository(pgPool)

	groupID := cfg.Kafka.GroupID
	if groupID == "" || groupID == "orders-consumer-group-1" {
		groupID = "ticket-service"
	}
	kafkaConsumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, groupID)
	defer kafkaConsumer.Close()

	consumerName := "ticket-service"
	logger.Info("Ticket Service Started", "consumer", consumerName, "group_id", groupID, "topic", cfg.Kafka.Topic, "brokers", cfg.Kafka.Brokers)

	for {
		msg, err := kafkaConsumer.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				logger.Info("Ticket Service stopping")
				break
			}
			logger.Error("failed to fetch message", "error", err)
			time.Sleep(1 * time.Second)
			continue
		}

		const maxRetries = 5
		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				backoff := time.Duration(1<<attempt) * time.Second
				logger.Info("Retry attempt", "attempt", attempt, "max", maxRetries, "backoff", backoff)
				time.Sleep(backoff)
			}

			processErr := func() error {
				var ev domainEvent.Message
				if err := json.Unmarshal(msg.Value, &ev); err != nil {
					logger.Error("failed to unmarshal event envelope", "error", err)
					return nil
				}

				if ev.Type != "PaymentAuthorized" {
					return nil
				}

				logger.Info("Received event", "type", ev.Type, "correlation_id", ev.CorrelationID, "event_id", ev.ID)

				var p paymentAuthorizedPayload
				if err := json.Unmarshal(ev.Payload, &p); err != nil {
					return fmt.Errorf("unmarshal PaymentAuthorized payload: %w", err)
				}

				tx, err := pgPool.Begin(ctx)
				if err != nil {
					return fmt.Errorf("begin tx: %w", err)
				}
				defer tx.Rollback(ctx)

				isNew, err := inboxRepo.SaveIfNotExists(ctx, tx, consumerName, ev.ID, ev.Type, ev.CorrelationID)
				if err != nil {
					return fmt.Errorf("inbox save: %w", err)
				}

				if !isNew {
					if err := tx.Commit(ctx); err != nil {
						return fmt.Errorf("commit noop tx: %w", err)
					}
					return nil
				}

				// Simulate load (2-3s) to show cascading steps in UI
				time.Sleep(2*time.Second + time.Duration(rand.Intn(1000))*time.Millisecond)

				ticketID := uuid.New().String()
				t := &ticket.Ticket{
					ID:         ticketID,
					OrderID:    p.OrderID,
					FromCity:   p.FromCity,
					ToCity:     p.ToCity,
					TravelDate: p.TravelDate,
					TravelTime: p.TravelTime,
					Airline:    p.Airline,
					Status:     "ISSUED",
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}

				ctxWithTx := context.WithValue(ctx, "tx", tx)
				if err := ticketRepo.Create(ctxWithTx, t); err != nil {
					return fmt.Errorf("create ticket: %w", err)
				}

				payload, err := json.Marshal(ticketIssuedPayload{OrderID: p.OrderID, TicketID: ticketID})
				if err != nil {
					return fmt.Errorf("marshal TicketIssued payload: %w", err)
				}

				outboxEvent := &outbox.Event{
					ID:            uuid.New().String(),
					EventType:     "TicketIssued",
					Payload:       payload,
					Status:        "new",
					CorrelationID: p.OrderID,
					CausationID:   ev.ID,
					Producer:      consumerName,
					CreatedAt:     time.Now(),
				}
				if err := outboxRepo.Create(ctxWithTx, outboxEvent); err != nil {
					return fmt.Errorf("create outbox event: %w", err)
				}

				if err := tx.Commit(ctx); err != nil {
					return fmt.Errorf("commit tx: %w", err)
				}

				ticketsProcessed.Inc()
				logger.Info("Ticket issued", "order_id", p.OrderID, "ticket_id", ticketID, "event_id", ev.ID)
				return nil
			}()

			if processErr == nil {
				if err := kafkaConsumer.CommitMessages(ctx, msg); err != nil {
					logger.Error("failed to commit kafka message", "error", err)
				}
				break
			}

			logger.Error("Processing failed", "error", processErr)
			if attempt == maxRetries {
				logger.Error("DLQ: Dropping message after retries", "retries", maxRetries, "error", processErr)
				if err := kafkaConsumer.CommitMessages(ctx, msg); err != nil {
					logger.Error("failed to commit drop to kafka", "error", err)
				}
			}
		}
	}
}

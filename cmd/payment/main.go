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
	"project/internal/domain/order"
	"project/internal/domain/outbox"
	"project/internal/domain/payment"
	"project/internal/infrastructure/kafka"
	"project/internal/infrastructure/postgres"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	paymentsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "payment_service_events_processed_total",
		Help: "The total number of processed events by payment service",
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
		logger.Info("Payment metrics listening on :9094")
		http.ListenAndServe(":9094", mux)
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
	paymentRepo := postgres.NewPaymentRepository(pgPool)

	groupID := cfg.Kafka.GroupID
	if groupID == "" || groupID == "orders-consumer-group-1" {
		groupID = "payment-service"
	}
	kafkaConsumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, groupID)
	defer kafkaConsumer.Close()

	consumerName := "payment-service"
	logger.Info("Payment Service Started", "consumer", consumerName, "group_id", groupID)

	for {
		msg, err := kafkaConsumer.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
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

				if ev.Type != "OrderCreated" {
					return nil
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

				var o order.Order
				if err := json.Unmarshal(ev.Payload, &o); err != nil {
					return fmt.Errorf("unmarshal order payload: %w", err)
				}

				// Simulate load (2-3s) to show cascading steps in UI
				time.Sleep(2*time.Second + time.Duration(rand.Intn(1000))*time.Millisecond)

				paymentID := uuid.New().String()
				p := &payment.Payment{
					ID:        paymentID,
					OrderID:   o.ID,
					Status:    "AUTHORIZED",
					Amount:    o.TotalAmount,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				ctxWithTx := context.WithValue(ctx, "tx", tx)
				if err := paymentRepo.Create(ctxWithTx, p); err != nil {
					return fmt.Errorf("create payment: %w", err)
				}

				payload, err := json.Marshal(paymentAuthorizedPayload{
					OrderID:    o.ID,
					PaymentID:  paymentID,
					Amount:     o.TotalAmount,
					FromCity:   o.FromCity,
					ToCity:     o.ToCity,
					TravelDate: o.TravelDate,
					TravelTime: o.TravelTime,
					Airline:    o.Airline,
				})
				if err != nil {
					return fmt.Errorf("marshal PaymentAuthorized payload: %w", err)
				}

				outboxEvent := &outbox.Event{
					ID:            uuid.New().String(),
					EventType:     "PaymentAuthorized",
					Payload:       payload,
					Status:        "new",
					CorrelationID: o.ID,
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

				paymentsProcessed.Inc()
				logger.Info("Payment authorized", "order_id", o.ID, "event_id", ev.ID, "payment_id", paymentID)
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

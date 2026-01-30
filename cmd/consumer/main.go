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
	"project/internal/infrastructure/kafka"
	"project/internal/infrastructure/postgres"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ordersProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "consumer_orders_processed_total",
		Help: "The total number of processed orders",
	})
	processingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "consumer_processing_duration_seconds",
		Help:    "Time taken to process order",
		Buckets: []float64{0.1, 0.5, 1, 2, 5},
	})
)

func main() {
	// Initialize structured JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load config
	cfg, err := config.New()
	if err != nil {
		logger.Error("Failed to load config, using defaults", "error", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Metrics Server
	// Metrics Server
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		logger.Info("Consumer metrics listening on :9091")
		http.ListenAndServe(":9091", mux)
	}()

	// Infrastructure (Postgres)
	infraFactory := infrastructure.NewFactory(cfg)
	defer infraFactory.Close()

	pgPool, err := infraFactory.Postgres(ctx)
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}

	inboxRepo := postgres.NewInboxRepository(pgPool)
	orderRepo := postgres.NewOrderRepository(pgPool)

	// Kafka Consumer
	groupID := cfg.Kafka.GroupID
	if groupID == "" {
		groupID = "order-service"
	}
	kafkaConsumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, groupID)
	defer kafkaConsumer.Close()

	consumerName := "order-service"
	logger.Info("Order Consumer Started", "consumer", consumerName, "group_id", groupID)

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

		// Retry Loop
		const maxRetries = 5
		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				backoff := time.Duration(1<<attempt) * time.Second
				logger.Info("Retry attempt", "attempt", attempt, "max", maxRetries, "backoff", backoff)
				time.Sleep(backoff)
			}

			processErr := func() error {
				started := time.Now()
				var ev domainEvent.Message
				if err := json.Unmarshal(msg.Value, &ev); err != nil {
					// Not our envelope (or corrupt). Commit and move on.
					logger.Error("failed to unmarshal event envelope", "error", err)
					return nil
				}

				switch ev.Type {
				case "PaymentAuthorized", "TicketIssued", "PaymentFailed":
					// handled below
				default:
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

				ctxWithTx := context.WithValue(ctx, "tx", tx)

				// Simulate load (2-3s) to make the saga feel cascading
				time.Sleep(2*time.Second + time.Duration(rand.Intn(1000))*time.Millisecond)

				switch ev.Type {
				case "PaymentAuthorized":
					if err := orderRepo.UpdateStatus(ctxWithTx, ev.CorrelationID, "PAYMENT_AUTHORIZED"); err != nil {
						return fmt.Errorf("update order status: %w", err)
					}
				case "TicketIssued":
					if err := orderRepo.UpdateStatus(ctxWithTx, ev.CorrelationID, "TICKET_ISSUED"); err != nil {
						return fmt.Errorf("update order status: %w", err)
					}
				case "PaymentFailed":
					if err := orderRepo.UpdateStatus(ctxWithTx, ev.CorrelationID, "CANCELLED"); err != nil {
						return fmt.Errorf("update order status: %w", err)
					}
				}

				if err := tx.Commit(ctx); err != nil {
					return fmt.Errorf("commit tx: %w", err)
				}

				processingDuration.Observe(time.Since(started).Seconds())
				ordersProcessed.Inc()
				logger.Info("Order state updated", "type", ev.Type, "correlation_id", ev.CorrelationID, "event_id", ev.ID)
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

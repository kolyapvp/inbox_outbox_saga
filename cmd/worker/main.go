package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"project/internal/application/factories/infrastructure"
	"project/internal/config"
	"project/internal/infrastructure/kafka"
	"project/internal/infrastructure/postgres"
	"project/internal/worker"
)

func main() {
	// Initialize structured JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.New()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger.Info(">>> STARTING NEW WORKER POLLER <<<")

	// Infrastructure
	infraFactory := infrastructure.NewFactory(cfg)
	defer infraFactory.Close()

	pgPool, err := infraFactory.Postgres(ctx)
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}

	// Dependencies
	outboxRepo := postgres.NewOutboxRepository(pgPool)

	kafkaProd := kafka.NewProducer(kafka.Config{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.Topic,
	})
	defer kafkaProd.Close()

	// Worker (Poller)
	w := worker.NewOutboxPoller(outboxRepo, kafkaProd)

	// Run
	if err := w.Run(ctx); err != nil {
		logger.Error("worker stopped with error", "error", err)
	}

	logger.Info("worker exited")
}

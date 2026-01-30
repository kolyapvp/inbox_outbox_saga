package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"project/internal/api"
	"project/internal/application/factories/infrastructure"
	"project/internal/config"
	"project/internal/grpc"
	"project/internal/infrastructure/postgres"
	redisInfra "project/internal/infrastructure/redis"
	"project/internal/usecase"
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

	infraFactory := infrastructure.NewFactory(cfg)
	defer infraFactory.Close()

	// Initialize dependencies
	pgPool, err := infraFactory.Postgres(ctx)
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}

	// Redis
	redisClient, err := redisInfra.NewClient(ctx, redisInfra.Config{
		Addr: cfg.Redis.Addr,
	})
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// Repositories
	orderRepo := postgres.NewOrderRepository(pgPool)
	outboxRepo := postgres.NewOutboxRepository(pgPool)
	inboxRepo := postgres.NewInboxRepository(pgPool)
	paymentRepo := postgres.NewPaymentRepository(pgPool)
	ticketRepo := postgres.NewTicketRepository(pgPool)
	txManager := postgres.NewTxManager(pgPool)

	// UseCases
	createOrderUC := usecase.NewCreateOrder(txManager, orderRepo, outboxRepo)
	getOrderUC := usecase.NewGetOrder(redisClient, orderRepo)
	getWorkflowUC := usecase.NewGetWorkflow(orderRepo, outboxRepo, inboxRepo, paymentRepo, ticketRepo)
	refundOrderUC := usecase.NewRefundOrder(txManager, orderRepo, outboxRepo)

	// gRPC Server (Mocked start)
	grpcService := grpc.NewServiceServer(createOrderUC)
	_ = grpcService // In real app: pb.RegisterOrderServiceServer(grpcServer, grpcService)

	// REST API Handler
	handlers := api.NewHandlers(createOrderUC, getOrderUC, getWorkflowUC, refundOrderUC)
	apiHandler := api.NewRouter(handlers, redisClient)

	srv := &http.Server{
		Addr:    ":" + cfg.HTTP.Port,
		Handler: apiHandler,
	}

	go func() {
		logger.Info("Server starting", "port", cfg.HTTP.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("listen failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server exiting")
}

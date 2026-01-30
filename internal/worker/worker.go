package worker

import (
	"context"
	"log"
	"time"

	"project/internal/config"
)

type Worker struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Worker {
	return &Worker{
		cfg: cfg,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	log.Println("Worker started")

	// Simulate work loop
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Process events from Kafka or DB
			log.Println("Worker processing...")
		}
	}
}

package worker

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"

	domainEvent "project/internal/domain/event"
	"project/internal/infrastructure/kafka"
	"project/internal/infrastructure/postgres"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	eventsPublished = promauto.NewCounter(prometheus.CounterOpts{
		Name: "worker_outbox_events_published_total",
		Help: "The total number of events published to Kafka",
	})
	publishErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "worker_outbox_publish_errors_total",
		Help: "The total number of failed publish attempts",
	})
)

type OutboxPoller struct {
	outboxRepo *postgres.OutboxRepository
	kafkaProd  *kafka.Producer
}

func NewOutboxPoller(outboxRepo *postgres.OutboxRepository, kafkaProd *kafka.Producer) *OutboxPoller {
	// Start metrics server for worker
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		log.Println("Worker metrics listening on :9093")
		http.ListenAndServe(":9093", mux)
	}()

	return &OutboxPoller{
		outboxRepo: outboxRepo,
		kafkaProd:  kafkaProd,
	}
}

func (p *OutboxPoller) Run(ctx context.Context) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Printf("OutboxPoller started (Topic: %s)", p.kafkaProd.GetTopic())

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := p.processBatch(ctx); err != nil {
				log.Printf("failed to process batch: %v", err)
			}
		}
	}
}

func (p *OutboxPoller) processBatch(ctx context.Context) error {
	events, err := p.outboxRepo.FetchBatch(ctx, 10)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	var processedIDs []string
	var failedIDs []string

	// Simulate load (2-3s) so the publish step is observable
	time.Sleep(2*time.Second + time.Duration(rand.Intn(1000))*time.Millisecond)

	for _, e := range events {
		log.Printf("Sending event %s to kafka...", e.ID)

		key := []byte(e.CorrelationID)
		if len(key) == 0 {
			key = []byte(e.ID)
		}

		msg := domainEvent.Message{
			ID:            e.ID,
			Type:          e.EventType,
			CorrelationID: e.CorrelationID,
			CausationID:   e.CausationID,
			Producer:      e.Producer,
			OccurredAt:    time.Now().UTC(),
			Payload:       e.Payload,
		}

		value, err := json.Marshal(msg)
		if err != nil {
			log.Printf("failed to marshal event %s: %v", e.ID, err)
			publishErrors.Inc()
			failedIDs = append(failedIDs, e.ID)
			continue
		}

		// Create a timeout context for this specific send operation
		sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = p.kafkaProd.SendMessage(sendCtx, key, value)
		cancel()

		if err != nil {
			log.Printf("failed to send event %s to kafka: %v", e.ID, err)
			publishErrors.Inc()
			failedIDs = append(failedIDs, e.ID)
			continue
		}

		log.Printf("Successfully sent event %s", e.ID)
		eventsPublished.Inc()
		processedIDs = append(processedIDs, e.ID)
	}

	if len(processedIDs) > 0 {
		log.Printf("Marking %d events as processed in DB...", len(processedIDs))
		if err := p.outboxRepo.MarkProcessed(ctx, processedIDs); err != nil {
			return err
		}
		log.Printf("Processed %d events", len(processedIDs))
	}

	if len(failedIDs) > 0 {
		if err := p.outboxRepo.MarkFailed(ctx, failedIDs); err != nil {
			log.Printf("failed to mark events as failed: %v", err)
		}
	}

	return nil
}

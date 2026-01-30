package kafka

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, topic string, groupID string) *Consumer {
	startOffset := kafka.FirstOffset
	// When a consumer group has no committed offset yet, kafka-go uses StartOffset.
	// For demo purposes it's often useful to start from the latest message.
	// Supported: "earliest" (default), "latest".
	if v := strings.TrimSpace(os.Getenv("KAFKA_START_OFFSET")); v != "" {
		switch strings.ToLower(v) {
		case "latest":
			startOffset = kafka.LastOffset
		case "earliest":
			startOffset = kafka.FirstOffset
		}
	}

	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: false, // Force IPv4
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		MinBytes:    1,    // Process immediately
		MaxBytes:    10e6, // 10MB
		MaxWait:     1 * time.Second,
		Dialer:      dialer,
		StartOffset: startOffset,
	})
	return &Consumer{reader: r}
}

func (c *Consumer) FetchMessage(ctx context.Context) (kafka.Message, error) {
	return c.reader.FetchMessage(ctx)
}

func (c *Consumer) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	return c.reader.CommitMessages(ctx, msgs...)
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

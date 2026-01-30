package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type Config struct {
	Brokers []string
	Topic   string
}

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(cfg Config) *Producer {
	w := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Brokers...),
		Topic:                  cfg.Topic,
		Balancer:               &kafka.Hash{},
		MaxAttempts:            5,
		ReadTimeout:            10 * time.Second,
		WriteTimeout:           10 * time.Second,
		Async:                  false,
		AllowAutoTopicCreation: true,
	}

	return &Producer{writer: w}
}

func (p *Producer) SendMessage(ctx context.Context, key, value []byte) error {
	err := p.writer.WriteMessages(ctx,
		kafka.Message{
			Key:   key,
			Value: value,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	return nil
}

func (p *Producer) GetTopic() string {
	return p.writer.Topic
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	"sentinel/internal/core/domain"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, topic string, groupID string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  groupID, // Identifies this worker group
			MinBytes: 10e3,    // 10KB
			MaxBytes: 10e6,    // 10MB
		}),
	}
}

func (c *Consumer) Start(ctx context.Context, handler func(ctx context.Context, event domain.AuditEvent) error) error {
	slog.Info("Starting Kafka Consumer...")

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.Error("Error fetching message", "err", err)
			continue
		}

		var event domain.AuditEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			slog.Error("Error unmarshalling event", "err", err)
			continue
		}

		if err := handler(ctx, event); err != nil {
			slog.Error("Handler failed", "err", err)
			continue
		}

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			slog.Error("Failed to commit message", "err", err)
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

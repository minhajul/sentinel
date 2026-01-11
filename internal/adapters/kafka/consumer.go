package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

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
	fmt.Println("Starting Kafka Consumer...")

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("Error fetching message: %v\n", err)
			continue
		}

		var event domain.AuditEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("Error unmarshalling event: %v\n", err)
			continue
		}

		if err := handler(ctx, event); err != nil {
			log.Printf("Handler failed: %v\n", err)
			continue
		}

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("Failed to commit message: %v\n", err)
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

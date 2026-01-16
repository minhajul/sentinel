package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"sentinel/internal/core/domain"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			BatchTimeout: 10 * time.Millisecond,
		},
	}
}

func (p *Producer) Publish(ctx context.Context, event domain.AuditEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.ActorID),
		Value: payload,
		Time:  time.Now(),
	})

	if err != nil {
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	return nil
}

func (p *Producer) Ping(ctx context.Context) error {
	client := &kafka.Client{
		Addr: p.writer.Addr,
	}
	_, err := client.Metadata(ctx, &kafka.MetadataRequest{
		Topics: []string{p.writer.Topic},
	})
	return err
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

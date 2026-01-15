package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sentinel/configs"
	postgres "sentinel/internal/adapters/postgresql"
	"syscall"
	"time"

	"sentinel/internal/adapters/kafka"
	"sentinel/internal/core/domain"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := configs.LoadConfig()

	repo, err := postgres.NewRepository(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Could not connect to DB: %v", err)
	}

	defer repo.Close()

	now := time.Now()
	nextMonth := now.AddDate(0, 1, 0)

	if err := repo.EnsurePartitionExists(ctx, now); err != nil {
		log.Printf("Warning: Failed to ensure current partition: %v", err)
	}
	if err := repo.EnsurePartitionExists(ctx, nextMonth); err != nil {
		log.Printf("Warning: Failed to ensure next partition: %v", err)
	}

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID)
	defer consumer.Close()

	eventHandler := func(ctx context.Context, event domain.AuditEvent) error {
		log.Printf("Saving event %s to DB...", event.EventID)
		return repo.Save(ctx, event)
	}

	log.Println("Consumer starting...")
	if err := consumer.Start(ctx, eventHandler); err != nil {
		log.Fatalf("Consumer failed: %v", err)
	}
}

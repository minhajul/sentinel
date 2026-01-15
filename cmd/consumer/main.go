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
	
	repo, _ := postgres.NewRepository(cfg.DatabaseURL)

	now := time.Now()
	nextMonth := now.AddDate(0, 1, 0)

	_ = repo.EnsurePartitionExists(ctx, now)
	_ = repo.EnsurePartitionExists(ctx, nextMonth)

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

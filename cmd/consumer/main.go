package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sentinel/configs"
	postgres "sentinel/internal/adapters/postgresql"
	"sentinel/pkg/logger"
	"syscall"
	"time"

	"sentinel/internal/adapters/kafka"
	"sentinel/internal/core/domain"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := configs.LoadConfig()
	logger.InitLogger(cfg.LokiURL)

	repo, err := postgres.NewRepository(cfg.DatabaseURL)
	if err != nil {
		slog.Error("Could not connect to DB", "err", err)
		os.Exit(1)
	}

	defer repo.Close()

	// Ensure partitions exist for current + next 2 months
	now := time.Now()
	for i := range 3 {
		t := now.AddDate(0, i, 0)
		if err := repo.EnsurePartitionExists(ctx, t); err != nil {
			slog.Warn("Failed to ensure partition", "month", t.Format("2006-01"), "err", err)
		} else {
			slog.Info("Partition ensured", "month", t.Format("2006-01"))
		}
	}

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID)
	defer consumer.Close()

	eventHandler := func(ctx context.Context, event domain.AuditEvent) error {
		slog.Info("Saving event to DB...", "eventID", event.EventID)
		return repo.Save(ctx, event)
	}

	slog.Info("Consumer starting...")
	if err := consumer.Start(ctx, eventHandler); err != nil {
		slog.Error("Consumer failed", "err", err)
		os.Exit(1)
	}
}

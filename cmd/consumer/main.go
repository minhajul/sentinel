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
	"sentinel/internal/core/monitoring"

	"net/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	// Ensure partitions exist for current + next 2 months
	now := time.Now()
	for i := range 3 {
		t := now.AddDate(0, i, 0)
		if err := repo.EnsurePartitionExists(ctx, t); err != nil {
			log.Printf("Warning: Failed to ensure partition for %s: %v", t.Format("2006-01"), err)
		} else {
			log.Printf("Partition ensured for %s", t.Format("2006-01"))
		}
	}

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID)
	defer consumer.Close()

	eventHandler := func(ctx context.Context, event domain.AuditEvent) error {
		start := time.Now()
		log.Printf("Saving event %s to DB...", event.EventID)
		
		err := repo.Save(ctx, event)
		
		status := "success"
		if err != nil {
			status = "error"
		}
		
		monitoring.ConsumerProcessingDuration.WithLabelValues(cfg.KafkaTopic, status).Observe(time.Since(start).Seconds())
		
		return err
	}

	go func() {
		log.Println("Starting metrics server on :2112...")
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":2112", nil); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics HTTP server failed: %v", err)
		}
	}()

	log.Println("Consumer starting...")
	if err := consumer.Start(ctx, eventHandler); err != nil {
		log.Fatalf("Consumer failed: %v", err)
	}
}

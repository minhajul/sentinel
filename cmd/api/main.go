package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sentinel/configs"
	"syscall"
	"time"

	"sentinel/internal/adapters/kafka"
	postgres "sentinel/internal/adapters/postgresql"
	"sentinel/internal/core/domain"
	"sentinel/internal/core/middlewares"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

func main() {
	cfg := configs.LoadConfig()

	producer := kafka.NewProducer(cfg.KafkaBrokers, "audit-logs")
	defer producer.Close()

	db, err := postgres.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	routing := chi.NewRouter()

	routing.Use(middleware.RequestID)
	routing.Use(middleware.RealIP)
	routing.Use(middleware.Logger)
	routing.Use(middleware.Recoverer)
	routing.Use(middleware.Timeout(60 * time.Second))

	eventsLimiter := middlewares.RateLimit(100, 1*time.Minute)

	// Root health endpoint
	routing.With(eventsLimiter).Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "UP",
			"data":    "Sentinel api is working.",
			"version": "1.0.0",
		})
	})

	// Readiness probe endpoint
	routing.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var dbStatus = "UP"
		if err := db.PingContext(ctx); err != nil {
			dbStatus = "DOWN"
		}

		var kafkaStatus = "UP"
		if err := producer.Ping(ctx); err != nil {
			kafkaStatus = "DOWN"
		}

		status := http.StatusOK
		if dbStatus == "DOWN" || kafkaStatus == "DOWN" {
			status = http.StatusServiceUnavailable
		}

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "UP",
			"checks": map[string]string{
				"postgres": dbStatus,
				"kafka":    kafkaStatus,
			},
		})
	})

	// Audit event ingestion endpoint
	routing.With(eventsLimiter).Post("/events", func(w http.ResponseWriter, r *http.Request) {
		var req domain.AuditEvent

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		req.EventID = uuid.New()
		req.Timestamp = time.Now()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := producer.Publish(ctx, req); err != nil {
			log.Printf("Failed to publish event: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":   "queued",
			"event_id": req.EventID.String(),
		})
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: routing,
	}

	go func() {
		log.Println("API listening on port 8080...")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down API...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

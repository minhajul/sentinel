package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sentinel/configs"
	"sentinel/pkg/logger"
	"syscall"
	"time"

	"sentinel/internal/adapters/kafka"
	postgres "sentinel/internal/adapters/postgresql"
	"sentinel/internal/core/domain"
	"sentinel/internal/core/middlewares"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := configs.LoadConfig()
	logger.InitLogger(cfg.LokiURL)

	producer := kafka.NewProducer(cfg.KafkaBrokers, "audit-logs")
	defer producer.Close()

	db, err := postgres.NewConnection(cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to DB", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	routing := chi.NewRouter()

	routing.Use(middleware.RequestID)
	routing.Use(middleware.RealIP)
	routing.Use(middleware.Logger)
	routing.Use(middleware.Recoverer)
	routing.Use(middleware.Timeout(60 * time.Second))
	routing.Use(middlewares.PrometheusMetrics)

	eventsLimiter := middlewares.RateLimit(100, 1*time.Minute)

	// Root health endpoint
	routing.Get("/", func(w http.ResponseWriter, r *http.Request) {
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
			slog.Error("Failed to publish event", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":   "queued",
			"event_id": req.EventID.String(),
		})
	})

	// Prometheus metrics endpoint
	routing.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    ":8080",
		Handler: routing,
	}

	go func() {
		slog.Info("API listening on port 8080...")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down API...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

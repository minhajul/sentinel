package configs

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config holds runtime configuration for both sentinel-api and sentinel-consumer.
// Values are populated from environment variables (and a local .env file, if present).
type Config struct {
	// ServerPort is the HTTP port the API binds to.
	ServerPort string `envconfig:"PORT" default:"8080"`

	// DatabaseURL is the PostgreSQL DSN used by the API (for /health) and the consumer.
	DatabaseURL string `envconfig:"DB_DSN" required:"true"`

	// KafkaBrokers is the list of bootstrap brokers. Comma-separated in the env (e.g. "b1:9092,b2:9092").
	KafkaBrokers []string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`

	// KafkaTopic is the topic events are produced to and consumed from.
	KafkaTopic string `envconfig:"KAFKA_TOPIC" default:"audit-logs"`

	// KafkaGroupID identifies the consumer group.
	KafkaGroupID string `envconfig:"KAFKA_GROUP_ID" default:"audit-group-1"`

	// LokiURL is the base URL of the Loki push API. Empty means stdout-only logging.
	LokiURL string `envconfig:"LOKI_URL"`
}

// LoadConfig reads configuration from the environment. A local .env file is loaded
// first if present, but missing or unreadable files are not fatal. Returns an error
// if any required field is missing or a value cannot be parsed.
func LoadConfig() (*Config, error) {
	// Best-effort .env load. We do not treat "file not found" as an error because
	// production deployments inject env vars directly.
	_ = godotenv.Load()

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// MustLoad is the convenience wrapper for process entry points. It calls log.Fatal
// on any load error so that main() stays a single line.
func MustLoad() *Config {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	return cfg
}

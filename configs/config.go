package configs

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort   string
	DatabaseURL  string
	KafkaBrokers []string
	KafkaTopic   string
	KafkaGroupID string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Println("No .env file found, relying on system env vars")
	}

	cfg := &Config{
		ServerPort:   getEnv("PORT", "8080"),
		DatabaseURL:  getEnv("DB_DSN", ""),
		KafkaBrokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
		KafkaTopic:   getEnv("KAFKA_TOPIC", "audit-logs"),
		KafkaGroupID: getEnv("KAFKA_GROUP_ID", "audit-group-1"),
	}

	// Fail Fast: Validation
	if cfg.DatabaseURL == "" {
		log.Fatal("CRITICAL: DB_DSN environment variable is required")
	}

	return cfg
}

// Helper to read env var with a fallback default value
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

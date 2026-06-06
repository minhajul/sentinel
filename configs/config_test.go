package configs

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// envVars lists every env var LoadConfig reads. Tests use this to snapshot
// the host environment and restore it on cleanup.
var envVars = []string{"PORT", "DB_DSN", "KAFKA_BROKERS", "KAFKA_TOPIC", "KAFKA_GROUP_ID", "LOKI_URL"}

// withCleanEnv unsets all config-related env vars for the duration of the test
// and restores the host values on cleanup.
func withCleanEnv(t *testing.T) {
	t.Helper()
	saved := make(map[string]*string, len(envVars))
	for _, k := range envVars {
		if v, ok := os.LookupEnv(k); ok {
			saved[k] = &v
		} else {
			saved[k] = nil
		}
		_ = os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for k, v := range saved {
			if v == nil {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, *v)
			}
		}
	})
}

func TestLoadConfig_Defaults(t *testing.T) {
	withCleanEnv(t)

	// DB_DSN is required and the host env may not have it; supply a minimal valid value
	// so we can assert everything else falls back to defaults.
	t.Setenv("DB_DSN", "postgres://u:p@h:5432/d?sslmode=disable")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned error, want nil: %v", err)
	}

	if cfg.ServerPort != "8080" {
		t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, "8080")
	}
	if cfg.KafkaTopic != "audit-logs" {
		t.Errorf("KafkaTopic = %q, want %q", cfg.KafkaTopic, "audit-logs")
	}
	if cfg.KafkaGroupID != "audit-group-1" {
		t.Errorf("KafkaGroupID = %q, want %q", cfg.KafkaGroupID, "audit-group-1")
	}
	if !reflect.DeepEqual(cfg.KafkaBrokers, []string{"localhost:9092"}) {
		t.Errorf("KafkaBrokers = %v, want [localhost:9092]", cfg.KafkaBrokers)
	}
	if cfg.LokiURL != "" {
		t.Errorf("LokiURL = %q, want empty", cfg.LokiURL)
	}
	if cfg.DatabaseURL == "" {
		t.Error("DatabaseURL is empty after explicit set")
	}
}

func TestLoadConfig_Overrides(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("PORT", "9090")
	t.Setenv("DB_DSN", "postgres://u:p@h:5432/d?sslmode=disable")
	t.Setenv("KAFKA_BROKERS", "b1:9092,b2:9092,b3:9092")
	t.Setenv("KAFKA_TOPIC", "events-prod")
	t.Setenv("KAFKA_GROUP_ID", "g-prod")
	t.Setenv("LOKI_URL", "http://loki.internal:3100")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned error, want nil: %v", err)
	}

	if cfg.ServerPort != "9090" {
		t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, "9090")
	}
	if cfg.DatabaseURL != "postgres://u:p@h:5432/d?sslmode=disable" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if !reflect.DeepEqual(cfg.KafkaBrokers, []string{"b1:9092", "b2:9092", "b3:9092"}) {
		t.Errorf("KafkaBrokers = %v", cfg.KafkaBrokers)
	}
	if cfg.KafkaTopic != "events-prod" {
		t.Errorf("KafkaTopic = %q", cfg.KafkaTopic)
	}
	if cfg.KafkaGroupID != "g-prod" {
		t.Errorf("KafkaGroupID = %q", cfg.KafkaGroupID)
	}
	if cfg.LokiURL != "http://loki.internal:3100" {
		t.Errorf("LokiURL = %q", cfg.LokiURL)
	}
}

func TestLoadConfig_MissingRequired(t *testing.T) {
	withCleanEnv(t)
	// DB_DSN is unset; LoadConfig must fail.

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("LoadConfig() = nil error, want error for missing required DB_DSN")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "db_dsn") {
		t.Errorf("error %q does not mention DB_DSN", err.Error())
	}
}

package domain

import (
	"time"

	"github.com/google/uuid"
)

type AuditEvent struct {
	EventID      uuid.UUID      `json:"event_id" db:"event_id"`
	Timestamp    time.Time      `json:"timestamp" db:"timestamp"`
	ActorID      string         `json:"actor_id" db:"actor_id"`
	Action       string         `json:"action" db:"action"`
	ResourceType string         `json:"resource_type" db:"resource_type"`
	ResourceID   string         `json:"resource_id" db:"resource_id"`
	Changes      map[string]any `json:"changes" db:"changes"`
	Metadata     map[string]any `json:"metadata" db:"metadata"`
}

package ports

import (
	"context"
	"sentinel/internal/core/domain"
)

type EventProducer interface {
	Publish(ctx context.Context, event domain.AuditEvent) error
	Close() error
}

type EventConsumer interface {
	Start(ctx context.Context, handler func(ctx context.Context, event domain.AuditEvent) error) error
}

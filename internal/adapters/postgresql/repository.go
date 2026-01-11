package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sentinel/internal/core/domain"

	_ "github.com/lib/pq"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Save(ctx context.Context, event domain.AuditEvent) error {
	const query = `
		INSERT INTO audit_logs (
			event_id, timestamp, actor_id, action, 
			resource_type, resource_id, changes, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`

	changesJSON, err := json.Marshal(event.Changes)
	if err != nil {
		return fmt.Errorf("failed to marshal changes: %w", err)
	}

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		event.EventID,
		event.Timestamp,
		event.ActorID,
		event.Action,
		event.ResourceType,
		event.ResourceID,
		changesJSON,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}

	return nil
}

func (r *Repository) FindByChange(ctx context.Context, key, value string) ([]domain.AuditEvent, error) {
	const query = `
		SELECT event_id, timestamp, actor_id, action, changes 
		FROM audit_logs 
		WHERE changes @> $1
        ORDER BY timestamp DESC
        LIMIT 100
	`

	searchCriteria := map[string]string{key: value}
	searchJSON, _ := json.Marshal(searchCriteria)

	rows, err := r.db.QueryContext(ctx, query, searchJSON)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []domain.AuditEvent
	for rows.Next() {
		var e domain.AuditEvent
		var changesData []byte

		if err := rows.Scan(&e.EventID, &e.Timestamp, &e.ActorID, &e.Action, &changesData); err != nil {
			return nil, err
		}

		_ = json.Unmarshal(changesData, &e.Changes)
		events = append(events, e)
	}

	return events, nil
}

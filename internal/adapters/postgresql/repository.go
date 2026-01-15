package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sentinel/internal/core/domain"
	"time"

	_ "github.com/lib/pq"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(dsn string) (*Repository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	return &Repository{
		db: db,
	}, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
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

func (r *Repository) EnsurePartitionExists(ctx context.Context, t time.Time) error {
	partitionName := fmt.Sprintf("audit_logs_%s", t.Format("2006_01"))

	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0) // First day of next month

	query := fmt.Sprintf(`
        CREATE TABLE IF NOT EXISTS %s 
        PARTITION OF audit_logs 
        FOR VALUES FROM ('%s') TO ('%s');
    `, partitionName, start.Format("2006-01-02"), end.Format("2006-01-02"))

	_, err := r.db.ExecContext(ctx, query)
	return err
}

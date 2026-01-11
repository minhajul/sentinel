-- Create the parent table
CREATE TABLE audit_logs
(
    event_id      UUID         NOT NULL,
    timestamp     TIMESTAMPTZ  NOT NULL,
    actor_id      VARCHAR(255) NOT NULL,
    action        VARCHAR(50)  NOT NULL,
    resource_type VARCHAR(50),
    resource_id   VARCHAR(255),
    changes       JSONB,
    metadata      JSONB,
    PRIMARY KEY (timestamp, event_id) -- 'timestamp' must be part of PK for partitioning
) PARTITION BY RANGE (timestamp);

-- Create a GIN index for fast JSON querying
CREATE INDEX idx_audit_changes ON audit_logs USING GIN (changes);

-- Create initial partitions (e.g., for current month and next)
CREATE TABLE audit_logs_2026_01 PARTITION OF audit_logs
    FOR VALUES FROM
(
    '2026-01-01'
) TO
(
    '2026-01-31'
);
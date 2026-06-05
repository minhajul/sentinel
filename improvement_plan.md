# Sentinel — Step-by-Step Improvement Plan

> Goal: turn Sentinel from a working single-node prototype into a **production-grade, multi-tenant, compliance-ready, multi-million-user** audit log service.

The plan is organized into **6 sequential phases**. Each phase delivers an independently shippable improvement. Phases can be parallelized once the foundations (Phase 0) are in place.

---

## Phase 0 — Foundations (Week 1–2)
*Make the codebase trustworthy before adding more features.*

### 0.1 Add a Test Suite
The repo currently has **zero tests**. This is the single biggest blocker for scaling safely.
- [ ] Add `go test ./...` discipline: every package in `internal/` and `cmd/` should have at least one unit test.
- [ ] Target: **≥70 % coverage** on `internal/core/**` and `internal/adapters/**`.

### 0.2 Linting, Formatting, Pre-commit
- [ ] Add `golangci-lint` with config: `govet`, `staticcheck`, `errcheck`, `gosec`, `gocritic`, `revive`, `gofmt`, `gofumpt`, `misspell`.
- [ ] Add a `Makefile` target `make lint`, `make test`, `make test-integration`, `make build-api`, `make build-consumer`.

### 0.3 Configuration Hardening
`configs/config.go` is too thin for production.
- [ ] Replace ad-hoc env parsing with a typed struct + `envconfig` or `viper` (validation, defaults, slices, durations).

### 0.4 Security Baseline
- [ ] Move DB password and Kafka credentials out of `docker-compose.yml` into `.env` / Docker secrets / k8s secrets.
- [ ] Add `Content-Security-Policy`, `X-Content-Type-Options`, `Referrer-Policy` headers via a Chi middleware.

---

## Phase 1 — Reliability & Throughput (Week 3–5)
*Make the pipeline lossless and fast enough for spikes.*

### 1.1 Producer Hardening (`internal/adapters/kafka/producer.go`)
- [ ] Enable idempotent producer (`RequiredAcks: kafka.RequireAll`, `Async: false` for the first hop or `Async: true` with a delivery-report channel).
- [ ] Set `BatchSize`, `BatchBytes`, `BatchTimeout` from config.
- [ ] Use a meaningful **partition key**: `actor_id` is okay, but consider `(tenant_id, actor_id)` for better locality.

### 1.2 Consumer Hardening (`internal/adapters/kafka/consumer.go`)
The current consumer:
- Uses `FetchMessage` + per-event synchronous DB write.
- Calls `continue` on **any** error → **silent data loss**.
- Never commits on success inside a transaction.

Fix:
- [ ] Replace `continue` on handler error with a **Dead-Letter Queue (DLQ)** topic (`audit-logs-dlq`) including original payload + failure reason + headers.
- [ ] Add an in-memory retry loop with exponential backoff (3 attempts) before sending to DLQ.
- [ ] Move DB writes to **batches**: read N messages, build a single `INSERT ... VALUES (...), (...), ...` (use `pgx.CopyFrom` for max throughput).
- [ ] Only commit Kafka offsets **after** the batch is durably persisted.
- [ ] Add a **graceful shutdown** that drains the in-flight batch on SIGTERM.
- [ ] Make the consumer **horizontally scalable** by running multiple replicas (Kafka consumer group already supports this — confirm via metrics).
- [ ] Add an **idempotency check**: enforce unique `(event_id)` at the DB level (currently missing — see Phase 1.4).

### 1.3 Repository Hardening (`internal/adapters/postgresql/repository.go`)
- [ ] **Stop calling `EnsurePartitionExists` per event** — this is an expensive DDL hit on the hot path. Move to:
  - A background job in the consumer that ensures partitions for current + next 2 months at startup and on a daily tick.
  - A separate CronJob in k8s for the same purpose.
- [ ] Switch DB driver from `lib/pq` to **`jackc/pgx/v5`** (5–10× faster, native binary protocol, better batching via `CopyFrom`).
- [ ] Use a transactional outbox pattern only if you need API-side durability (currently the API is fire-and-forget — document the trade-off).
- [ ] Add a connection pool with `pgxpool`, expose pool stats to Prometheus.

### 1.4 Schema Improvements
- [ ] Add `UNIQUE (event_id)` constraint (idempotency).
- [ ] Add `tenant_id VARCHAR(64) NOT NULL` to support multi-tenant.
- [ ] Add a **chain hash** column `prev_hash BYTEA`, `hash BYTEA` to make logs **tamper-evident** (each event hash = `sha256(prev_hash || event_payload)`). This is the single most important compliance feature missing.
- [ ] Add composite indexes:
  - `(actor_id, timestamp DESC)`
  - `(resource_type, resource_id, timestamp DESC)`
  - `(tenant_id, timestamp DESC)`
  - `(tenant_id, action, timestamp DESC)`
- [ ] Replace single `idx_audit_changes` GIN index with **per-tenant** or **per-action** GIN indexes if data skews heavily.
- [ ] Move partitioning key to `(tenant_id, timestamp)` so each tenant gets its own range of partitions (improves query planning and per-tenant archival).
- [ ] Add a `pg_partman` or custom job that **automatically creates** monthly partitions and **detaches/archives** old ones to S3/GCS as Parquet files (cold storage).

---

## Phase 2 — Query API & Multi-Tenancy (Week 6–8)
*Make the data actually queryable and safe to share.*

### 2.1 Read API
The current API has no read endpoints. Add:
- [ ] `GET /v1/events` — list events with filters: `tenant_id`, `actor_id`, `action`, `resource_type`, `resource_id`, `from`, `to`, cursor pagination (no offset).
- [ ] `GET /v1/events/{event_id}` — single event lookup.
- [ ] `GET /v1/events/stream` — **SSE/WebSocket** for live tailing (useful for "time-travel debugging").
- [ ] `POST /v1/events/search` — arbitrary JSONB `changes` predicates (e.g. `{ "status": "failed" }`).
- [ ] `GET /v1/actors/{id}/timeline` — aggregate view per actor.

All responses must be **paged** with opaque cursors (`{ts, event_id}` pairs) — never `OFFSET`.

### 2.2 Bulk Ingest
- [ ] `POST /v1/events:batch` — accept an array of up to 1000 events, publish to Kafka in a single `WriteMessages` call. Cuts HTTP overhead ~100×.

### 2.3 Multi-Tenancy
- [ ] Resolve `tenant_id` from API key / JWT claim, not from request body (security: never trust the client).
- [ ] Enforce tenant isolation in the repository (every query has `WHERE tenant_id = $1`).
- [ ] Add per-tenant rate limits (`go-chi/httprate` supports custom key functions).
- [ ] Add per-tenant quota tracking (events/day) with Prometheus metrics and 429 responses.

### 2.4 API Versioning
- [ ] Move everything under `/v1/...`. Keep `/events` as a deprecated alias for one release.

---

## Phase 3 — Observability & Compliance (Week 9–10)

### 3.1 Tracing
- [ ] Add **OpenTelemetry** SDK: `go.opentelemetry.io/otel`.
- [ ] Instrument: HTTP server, HTTP client (Kafka), Postgres (`otelpgx`), Kafka producer/consumer (use `otelkafkago` or manual propagation).
- [ ] Propagate `traceparent` through Kafka message headers.
- [ ] Ship traces to **Tempo** (or any OTLP backend), correlate with logs in Grafana.

### 3.2 Metrics (extend `internal/core/monitoring/metrics.go`)
- [ ] Add: `events_ingested_total{tenant, action}`, `events_persisted_total{tenant, action}`, `events_dlq_total{reason}`, `ingest_latency_seconds`, `consumer_lag`, `db_pool_active_conns`, `kafka_producer_error_total`.
- [ ] Define **SLOs** and **SLIs** in a `SLO.md`:
  - Ingest availability ≥ 99.95 %.
  - End-to-end p99 ingest → persisted ≤ 5 s.
  - DLQ rate ≤ 0.01 %.

### 3.3 Alerting
- [ ] Add Prometheus alert rules (`configs/prometheus/alerts.yml`):
  - DLQ rate > threshold (5 min).
  - Consumer lag > N (10 min).
  - API 5xx rate > 1 %.
  - Disk usage on Postgres > 80 %.
- [ ] Wire alerts to **Alertmanager** → Slack/PagerDuty.

### 3.4 Dashboards
- [ ] Provision Grafana dashboards via JSON files in `configs/grafana/dashboards/` (ingestion rate, consumer lag, DB performance, error budget burn).
- [ ] Commit the dashboards to git — no manual clicks in the UI.

---

## Phase 4 — Horizontal Scale (Week 11–16)
*Now the system is ready to be deployed at scale.*

### 4.1 Kubernetes Manifests
Replace Docker Compose for production:
- [ ] `deploy/k8s/` with Helm chart or Kustomize.
- [ ] `Deployment` for `sentinel-api` and `sentinel-consumer` with HPA on CPU + custom metric (`kafka_producer_queue_depth` or RPS).
- [ ] `StatefulSet` for Postgres (or use a managed service).
- [ ] `StatefulSet` for Kafka (or use a managed service like MSK / Confluent Cloud).
- [ ] PodDisruptionBudgets, NetworkPolicies, PodSecurityStandards (`restricted`).
- [ ] Topology spread constraints across AZs.

### 4.2 Database Scale
- [ ] Add **connection pooler** (PgBouncer or `pgcat`) in front of Postgres to survive 10 K+ connections from many consumer replicas.

---

## Phase 5 — Advanced Features (Week 17+)

### 5.1 Tamper-Evidence End-to-End
- [ ] Implement the chain-hash column from 1.4; expose `GET /v1/verify?from=…&to=…` that re-computes the chain and returns the first broken link.
- [ ] Publish a daily **Merkle root** to an external witness (e.g. write to a public blockchain, or to a second cloud's S3 with Object Lock).

### 5.2 Streaming Analytics
- [ ] Pipe the Kafka topic to **ClickHouse** / **Materialize** / **Apache Pinot** for OLAP queries.
- [ ] Expose aggregations: events per actor per hour, top actions, error rates.

### 5.3 Integrations
- [ ] Webhook subscriptions: `POST /v1/webhooks` → fan out matching events to external URLs (sign with HMAC).
- [ ] gRPC ingestion endpoint (lower overhead for internal services).
- [ ] SDKs: Go, TypeScript, Python (auto-generated from the OpenAPI spec).

### 5.4 Schema-on-Read
- [ ] Versioned schemas per `(action, resource_type)`. Store `schema_version` on the event. Validate on write; expose a "schema registry" UI.

### 5.5 ML / Anomaly Detection
- [ ] Train an anomaly detection model on the audit stream (sudden spike of `login_failed` from one IP, etc.) and emit alerts.
- [ ] Use **Feature Store** (Feast) for serving features in real time.

### 5.6 Zero-Trust Security
- [ ] Replace static API keys with **OIDC** (e.g. Auth0, Keycloak) — validate JWT in the API middleware.
- [ ] **mTLS** between API ↔ Kafka and consumer ↔ Postgres (or use IAM auth).
- [ ] **Vet** dependencies on every build (`govulncheck`, `trivy fs`).
- [ ] Enable **Kyverno** / **OPA Gatekeeper** in the cluster to enforce signed images only.

---

## Capacity Targets (SLOs)

| Metric                       | Today        | Target @ 1 M users | Target @ 10 M users |
| ---------------------------- | ------------ | ------------------ | -------------------- |
| Sustained ingest             | ~hundreds/s  | 20 K/s             | 100 K/s              |
| Peak ingest burst (1 min)    | unbounded    | 100 K/s            | 500 K/s              |
| End-to-end p99 latency       | unknown      | < 5 s              | < 10 s               |
| Hot retention                | 1 month      | 90 d (Postgres)    | 30 d (PG) + S3       |
| Query p95 (last 24 h)        | n/a          | < 300 ms           | < 500 ms             |
| Storage per month            | unknown      | ~5 TB @ 20 K/s     | ~25 TB @ 100 K/s     |
| Availability                 | n/a          | 99.95 %            | 99.9 %               |
| Data loss budget             | unbounded    | 0 (with DLQ < 0.01%) | 0 (with DLQ < 0.01%) |

---

## Suggested First 30 Days

If you can only ship one thing per week, follow this order — it gives the biggest leverage:

1. **Week 1** — Tests + CI + lint (Phase 0). Everything else is built on top of this.
2. **Week 2** — DLQ + consumer batching + partition ensure off hot path (Phase 1.1–1.3).
3. **Week 3** — Add `tenant_id` + chain hash + indexes + migrate to `pgx` (Phase 1.4).
4. **Week 4** — Read API v1 (Phase 2.1) — unlock real users.

After 30 days, Sentinel will already be safe to put in front of internal traffic. Phase 4 work is what unlocks "millions of users".

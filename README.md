## Sentinel: Scalable Distributed Audit Log Service

**Sentinel** is a high-throughput, append-only event sourcing system designed to provide compliance, traceability, and "
time-travel" debugging capabilities for enterprise microservices.

It ingests structured business events via a REST API, buffers them through a Kafka stream to handle backpressure, and
persists them into a partitioned PostgreSQL database for efficient historical querying.

---

### Key Features

* **Asynchronous Ingestion:** Fire-and-forget API design (`202 Accepted`) ensures the logging system never blocks the
  main application.
* **Backpressure Handling:** Uses **Kafka** to decouple producers from consumers, allowing the system to absorb massive
  traffic spikes without data loss.
* **Database Partitioning:** PostgreSQL tables are partitioned by month. This allows for efficient querying of recent
  data and O(1) archival/deletion of old data (dropping partitions vs. expensive `DELETE` rows).
* **JSONB Indexing:** specific changes (`before`/`after` states) are stored in `JSONB` columns with GIN indexing,
  allowing for deep queries like "Who changed the *status* field to *failed*?".
* **Hexagonal Architecture:** The codebase strictly separates business logic (Domain) from infrastructure (Adapters),
  making it highly testable and maintainable.

### Tech Stack

* **Language:** Go (Golang) 1.23
* **Streaming:** Apache Kafka & Zookeeper
* **Database:** PostgreSQL 15 (with Range Partitioning)
* **Infrastructure:** Docker & Docker Compose
* **Libraries:** `segmentio/kafka-go`, `lib/pq` (Pure Go drivers)

### Prerequisites

* Docker & Docker Compose
* Make (optional, but recommended)

### Clone and Spin Up

```bash
git clone https://github.com/minhajul/sentinel.git
cd sentinel

# Start Kafka, Zookeeper, Postgres, API, and Consumer
make build

# OR

docker-compose up -d --build
```

### Initialize Database

The database needs the initial schema and partitions created.

```bash
# Add table and prevent modification
make migrate

# OR

# Connect to the running Postgres container
docker exec -it sentinel_postgres psql -U user -d sentinel_db

# Paste the contents of /migrations
```

### Test the Pipeline

**Send an Event (Producer):**

```bash
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "actor_id": "user_123",
    "action": "billing_update",
    "resource_type": "invoice",
    "resource_id": "inv_999",
    "changes": {"status": "paid", "amount": 500},
    "metadata": {"ip": "192.168.1.50"}
  }'
```

**Verify Storage (Consumer `->` DB):**
Check the logs to see the Consumer picking it up:

```bash
docker logs -f sentinel_consumer
```

### Engineering Decisions (Why I did this?)

#### Why Kafka instead of direct DB writes?

Direct writes couple the availability of the logging service to the database. If the DB slows down under load, the main
application hangs. By using Kafka, we introduce a buffer. The API accepts the request instantly (`202 Accepted`) and
offloads the heavy write operation to a background worker. This provides **resilience** and **load smoothing**.

#### Why Table Partitioning?

Audit logs grow indefinitely. Indexing a table with 100M+ rows degrades write and read performance. By partitioning by *
*Month**:

* **Writes** go to a smaller, hot table (e.g., `audit_logs_2024_01`).
* **Reads** for recent data are faster.
* **Archival** becomes a metadata operation (Detach Partition) rather than a row-by-row `DELETE` which causes table
  locking and vacuum issues.

#### Why JSONB for `changes`?

Audit logs vary wildly in structure. An `invoice` change has different fields than a `user_profile` change. A rigid SQL
schema would require hundreds of nullable columns or EAV (Entity-Attribute-Value) which is slow. PostgreSQL `JSONB` with
**GIN Indexing** gives us the flexibility of NoSQL (MongoDB-like) with the ACID compliance of a relational DB.

### License

MIT
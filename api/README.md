# X-Ray API

Golang backend for X-Ray reasoning-based observability system.

## Architecture

```
api/
├── cmd/server/main.go       # Entry point
├── internal/
│   ├── handlers/            # HTTP handlers
│   │   ├── ingest.go        # POST endpoints for traces/events/decisions
│   │   └── query.go         # GET endpoints for querying
│   ├── models/              # Data models
│   │   ├── models.go        # Core types (Trace, Event, Decision)
│   │   └── requests.go      # API request/response types
│   └── store/               # Database abstraction
│       ├── store.go         # Store interface (abstract)
│       └── dynamodb/        # DynamoDB implementation
└── Makefile
```

## Abstract Database Layer

The `store.Store` interface abstracts all database operations:

```go
type Store interface {
    // Trace operations
    CreateTrace(ctx context.Context, trace *models.Trace) error
    GetTrace(ctx context.Context, traceID string) (*models.Trace, error)
    GetTraceWithEvents(ctx context.Context, traceID string) (*models.TraceWithEvents, error)
    // ... more methods
}
```

To add a new database (MongoDB, PostgreSQL, Cassandra):

1. Create `internal/store/mongodb/mongodb.go`
2. Implement the `Store` interface
3. Update `main.go` to instantiate your new store

## Quick Start

### Prerequisites

- Go 1.21+
- Docker (for local DynamoDB)
- AWS CLI (optional, for table creation)

### Run Locally

```bash
# Start local DynamoDB
make dynamodb-local

# Create table
make create-table

# Run server
make run-local
```

Server starts at `http://localhost:8080`
Swagger UI at `http://localhost:8080/swagger/index.html`

### Environment Variables

| Variable            | Default   | Description        |
| ------------------- | --------- | ------------------ |
| `PORT`              | 8080      | Server port        |
| `DYNAMODB_ENDPOINT` | (AWS)     | Local DynamoDB URL |
| `DYNAMODB_TABLE`    | xray_data | Table name         |
| `AWS_REGION`        | us-east-1 | AWS region         |

## API Endpoints

### Ingest

| Method | Endpoint                  | Description            |
| ------ | ------------------------- | ---------------------- |
| POST   | `/api/v1/traces`          | Create trace           |
| POST   | `/api/v1/traces/batch`    | Batch create traces    |
| PATCH  | `/api/v1/traces/{id}`     | Update trace           |
| POST   | `/api/v1/events`          | Create event           |
| POST   | `/api/v1/events/batch`    | Batch create events    |
| POST   | `/api/v1/decisions`       | Create decision        |
| POST   | `/api/v1/decisions/batch` | Batch create decisions |

### Query

| Method | Endpoint                                     | Description           |
| ------ | -------------------------------------------- | --------------------- |
| GET    | `/api/v1/traces`                             | List traces           |
| GET    | `/api/v1/traces/{id}`                        | Get trace with events |
| GET    | `/api/v1/traces/{id}/events`                 | Get events for trace  |
| GET    | `/api/v1/traces/{id}/events/{eid}/decisions` | Get decisions         |
| GET    | `/api/v1/items/{id}/history`                 | Item decision history |
| GET    | `/api/v1/query/events`                       | Query events          |
| POST   | `/api/v1/query`                              | Advanced query        |
| GET    | `/health`                                    | Health check          |

## Example Requests

### Create Trace

```bash
curl -X POST http://localhost:8080/api/v1/traces \
  -H "Content-Type: application/json" \
  -d '{
    "pipeline_id": "competitor-selection",
    "started_at": "2024-01-05T10:00:00Z",
    "metadata": {"source_asin": "B08N5W"}
  }'
```

### Create Event

```bash
curl -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{
    "trace_id": "<trace_id>",
    "step_name": "filter_price",
    "step_type": "filter",
    "input_count": 1847,
    "output_count": 312,
    "started_at": "2024-01-05T10:00:01Z",
    "metrics": {"reduction_ratio": 0.831}
  }'
```

### Query Filter Steps with High Reduction

```bash
curl "http://localhost:8080/api/v1/query/events?step_type=filter&min_reduction_ratio=0.9"
```

### Get Item History

```bash
curl "http://localhost:8080/api/v1/items/ASIN-B08N5W/history"
```

## DynamoDB Schema

Production-grade single-table design with hierarchical keys (v2.0):

### Base Table

| PK | SK | Entity |
| --- | --- | --- |
| `TRACE#<id>` | `TRACE#<date>#<time>` | Trace metadata |
| `TRACE#<id>` | `EVENT#<date>#<time>#<event_id>` | Event under trace |
| `TRACE#<id>` | `DEC#<event_id>#<date>#<time>#<decision_id>` | Decision under trace |

**Key Benefits:**
- ✅ Single query retrieves complete trace with all events and decisions
- ✅ Natural time-based ordering within each trace
- ✅ Efficient pagination support
- ✅ No hot partitions (each trace is separate partition)

### Global Secondary Indexes

| Index | Purpose | PK | SK | Projection |
| ----- | ------- | -- | -- | ---------- |
| **GSI2** | Query traces by pipeline + time | `PIPELINE#<pipeline_id>` | `<date>#<time>#<trace_id>` | INCLUDE (status, metadata, tags) |
| **GSI3** | Cross-pipeline event analytics | `STEP#<step_type>` or `STEP#<step_type>#<pipeline_id>` | `<reduction_ratio>#<date>#<event_id>` | INCLUDE (metrics, counts) |
| **GSI4** | Event → Decisions with outcome filtering | `EVENT#<event_id>` | `<outcome>#<timestamp>#<decision_id>` | INCLUDE (item_id, reason_code, scores) |
| **GSI5** | Item history tracking (SPARSE) | `ITEM#<item_id>` | `<date>#<trace_id>#<event_id>#<decision_id>` | INCLUDE (outcome, reason_code) |

**GSI3 Dual-Write Strategy:** Each event writes twice:
1. Global: `STEP#filter` - for cross-pipeline analytics
2. Pipeline-specific: `STEP#filter#competitor-selection` - for pipeline-specific queries

**GSI5 Sparse Indexing:** Only indexes:
- ALL rejected decisions (for debugging)
- 1% of accepted decisions (deterministic sampling via SHA256)

**Storage Efficiency:** ~2.3× multiplier (vs 5× in naive design) = **54% cost savings**

## Testing

```bash
# Run tests
make test

# With coverage
make test-cover
```

## License

MIT

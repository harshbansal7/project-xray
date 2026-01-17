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
│       └── clickhouse/      # ClickHouse implementation
└── Makefile
```

## Database Layer

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

Currently uses ClickHouse for high-performance columnar storage.

## Quick Start

### Run with Docker

```bash
# From project root
make run
```

Server starts at `http://localhost:8080`

### Environment Variables

| Variable              | Default   | Description     |
| --------------------- | --------- | --------------- |
| `PORT`                | 8080      | Server port     |
| `CLICKHOUSE_HOST`     | localhost | ClickHouse host |
| `CLICKHOUSE_PORT`     | 9000      | ClickHouse port |
| `CLICKHOUSE_DATABASE` | xray      | Database name   |

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

### Query Filter Steps with High Reduction

```bash
curl "http://localhost:8080/api/v1/query/events?step_type=filter&min_reduction_ratio=0.9"
```

### Get Item History

```bash
curl "http://localhost:8080/api/v1/items/ASIN-B08N5W/history"
```

## ClickHouse Schema

Three main tables with columnar storage:

- `xray_traces` - Pipeline executions (ReplacingMergeTree for deduplication)
- `xray_events` - Step events with metrics
- `xray_decisions` - Item-level decisions with bloom filter on item_id

All tables have 90-day TTL for automatic data expiration.

## Testing

```bash
make test
```

## License

MIT

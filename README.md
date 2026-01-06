# X-Ray: Decision Observability for Multi-Step Pipelines

X-Ray provides visibility into _why_ decisions were made in non-deterministic systems. Unlike traditional tracing (Jaeger/Zipkin) which captures what happened, X-Ray captures the reasoning behind each decision.

## Quick Start

### 1. Start the Backend

```bash
docker-compose up
```

This starts:

- DynamoDB Local (port 8000)
- X-Ray API (port 8080)
- DynamoDB Admin UI (port 8001) - optional

### 2. Install the SDK

```bash
cd sdk/python
pip install -e .
```

### 3. Instrument Your Pipeline

```python
import xray_sdk as xray
from xray_sdk import XRayStepType, XRayPipelineID, XRayReasonCode

# Define your types
class MyPipelines(XRayPipelineID):
    SEARCH = "search"

class MySteps(XRayStepType):
    FILTER = "filter"
    LLM = "llm"

class MyReasons(XRayReasonCode):
    PRICE_TOO_HIGH = "PRICE_TOO_HIGH"

# Configure
xray.configure(endpoint="http://localhost:8080/api/v1")
xray.register_pipeline(MyPipelines.SEARCH, MySteps, MyReasons)

# Trace your pipeline
with xray.trace(MyPipelines.SEARCH) as t:
    with t.event("filter", step_type=MySteps.FILTER, capture="full") as e:
        e.set_input(candidates)
        for item in candidates:
            if item.price > 100:
                e.record_decision(item.id, "rejected",
                    reason_code=MyReasons.PRICE_TOO_HIGH,
                    scores={"price": item.price})
        e.set_output(filtered)
```

### 4. Query Your Data

```bash
# Get a trace
curl http://localhost:8080/api/v1/traces/{trace_id}

# Find filter steps with >90% reduction
curl 'http://localhost:8080/api/v1/query/events?step_type=filter&min_reduction_ratio=0.9'

# Get decisions for an event
curl http://localhost:8080/api/v1/traces/{trace_id}/events/{event_id}/decisions
```

## Project Structure

```
├── api/                 # Go API server
│   ├── cmd/server/      # Entry point
│   ├── internal/
│   │   ├── handlers/    # HTTP handlers (ingest + query)
│   │   ├── models/      # Data models
│   │   └── store/       # DynamoDB implementation
│   └── Makefile
├── sdk/python/          # Python SDK
│   ├── xray_sdk/
│   │   ├── trace.py     # Trace context manager
│   │   ├── event.py     # Event + decision recording
│   │   ├── client.py    # Async HTTP client
│   │   └── config.py    # Configuration + registry
│   └── examples/
├── docker-compose.yml   # Full stack setup
└── ARCHITECTURE.md      # Design decisions + trade-offs
```

## Approach

**Three-tier data model:**

- **Trace**: Complete pipeline execution
- **Event**: Single step (filter, LLM call, transform)
- **Decision**: Per-item outcome with reason code

**SDK design:**

- Type-safe: Developers define step types and reason codes as enums
- Non-blocking: Async batched sending by default
- Graceful degradation: Pipeline never fails if X-Ray backend is down

**Storage:**

- DynamoDB single-table design with 4 GSIs for efficient queries
- No table scans—all queries use indexes
- TTL-based auto-expiration (90 days)

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed design rationale.

## Known Limitations

1. **No authentication**: API is currently open. Production would need API keys + RBAC.

2. **No UI**: Query via API/curl only. A debug dashboard would help visualization.

3. **Python SDK only**: Go/TypeScript SDKs not implemented yet.

4. **Limited query patterns**: Complex queries (e.g., "decisions where score.price > 100") require client-side filtering.

5. **Large item snapshots**: Storing full item state can get expensive. Consider compression or tiered storage.

## Future Improvements

- Authentication and multi-tenancy
- Web dashboard for trace visualization
- Additional SDKs (Go, TypeScript)
- Compression for large payloads
- Natural language querying
- Alerting on anomalies (e.g., reduction ratio drops)

# X-Ray Architecture Document

## System Overview

X-Ray is a reasoning-based observability system for multi-step decision pipelines. Unlike traditional tracing (Jaeger/Zipkin) which captures _what_ happened, X-Ray captures _why_ decisions were made—enabling debugging of non-deterministic systems like LLM pipelines.

![X-Ray System Architecture](https://res.cloudinary.com/diwvz2qok/image/upload/v1767651208/Mermaid_Chart_-_Create_complex_visual_diagrams_with_text.-2026-01-05-221313_udj3kg.png)

---

## Data Model

### Core Entities

![DB Design](https://res.cloudinary.com/diwvz2qok/image/upload/v1767651241/Mermaid_Chart_-_Create_complex_visual_diagrams_with_text.-2026-01-05-221226_ypgfgc.png)

### DynamoDB Single-Table Design

All entities live in one table (`xray_data`) with composite keys:

| Entity   | PK                 | SK                             | Purpose            |
| -------- | ------------------ | ------------------------------ | ------------------ |
| Trace    | `TRACE#<trace_id>` | `TRACE#META`                   | Trace metadata     |
| Event    | `TRACE#<trace_id>` | `EVENT#<event_id>`             | Steps within trace |
| Decision | `TRACE#<trace_id>` | `DEC#<event_id>#<decision_id>` | Per-item decisions |

**GSI Strategy (query patterns without scans):**

| GSI  | PK                        | SK                        | Use Case                      |
| ---- | ------------------------- | ------------------------- | ----------------------------- |
| GSI2 | `GSI2_PIPE#<pipeline_id>` | `<trace_id>`              | Traces by pipeline            |
| GSI3 | `GSI3_STEP#<step_type>`   | `<event_id>`              | Cross-pipeline step analytics |
| GSI4 | `GSI4_EVT#<event_id>`     | `<outcome>#<decision_id>` | Decisions by event+outcome    |
| GSI5 | `GSI5_ITEM#<item_id>`     | `<decision_id>`           | Item history (sparse)         |

---

## Data Model Rationale

**Why hierarchical single-table design?**

I chose DynamoDB's single-table pattern because X-Ray's access patterns are predictable:

1. Get full trace → Single query with `PK = TRACE#<id>` fetches trace + events + decisions
2. Analytics by step type → GSI3 enables "find all filter steps" without knowing pipeline
3. Item debugging → GSI5 answers "where has this item appeared before?"

**Alternatives considered:**

- _Separate tables per entity_: Would require N+1 queries to fetch a trace with events. Rejected for latency.
- _Relational DB (Postgres)_: Better for ad-hoc joins, but harder to scale writes. Decision recording during filtering (5000+ items) would bottleneck.
- _Time-series DB_: Good for metrics, poor for structured decision trees.

**What breaks with different choices:**

- _Without GSI3_: Cross-pipeline queries like "all filter steps with >90% reduction" become expensive table scans
- _Without composite SK_: Can't efficiently query decisions for a specific event
- _Without UUIDv7_: Lose natural time-ordering; would need separate sort keys for temporal queries

---

## API Specification

### Ingest Endpoints

| Method  | Endpoint                   | Purpose                      |
| ------- | -------------------------- | ---------------------------- |
| `POST`  | `/api/v1/traces`           | Create trace                 |
| `PATCH` | `/api/v1/traces/{traceId}` | Update trace (complete/fail) |
| `POST`  | `/api/v1/traces/batch`     | Batch create traces          |
| `POST`  | `/api/v1/events`           | Create event                 |
| `POST`  | `/api/v1/events/batch`     | Batch create events          |
| `POST`  | `/api/v1/decisions`        | Create decision              |
| `POST`  | `/api/v1/decisions/batch`  | Batch create decisions       |

**Example: Create Event**

```json
POST /api/v1/events
{
  "trace_id": "019abc...",
  "step_name": "filter_products",
  "step_type": "filter",
  "capture_mode": "full",
  "input_count": 5000,
  "output_count": 30,
  "started_at": "2026-01-06T10:00:00Z"
}
```

### Query Endpoints

| Method | Endpoint                                                               | Purpose                           |
| ------ | ---------------------------------------------------------------------- | --------------------------------- |
| `GET`  | `/api/v1/traces/{traceId}`                                             | Get trace with events & decisions |
| `GET`  | `/api/v1/traces?pipeline_id=X`                                         | Query traces by pipeline          |
| `GET`  | `/api/v1/query/events?step_type=filter&min_reduction_ratio=0.9`        | Analytics queries                 |
| `GET`  | `/api/v1/traces/{traceId}/events/{eventId}/decisions?outcome=rejected` | Decisions for event               |
| `GET`  | `/api/v1/items/{itemId}/history`                                       | Item decision history             |

**Example: Query high-reduction filter steps**

```
GET /api/v1/query/events?step_type=filter&min_reduction_ratio=0.9

Response:
{
  "results": [
    {
      "event_id": "...",
      "step_name": "price_filter",
      "metrics": { "input_count": 5000, "output_count": 30, "reduction_ratio": 0.994 }
    }
  ],
  "count": 15,
  "next_cursor": "..."
}
```

---

## Debugging Walkthrough

**Scenario:** Phone case matched against laptop stand.

1. **Get the trace:**

   ```
   GET /api/v1/traces/{bad_trace_id}
   ```

   Response shows all events in order: `generate_keywords` → `search` → `filter` → `llm_rank` → `select`

2. **Check keyword generation (first potential failure point):**

   ```json
   {
     "step_name": "generate_keywords",
     "step_type": "llm",
     "annotations": {
       "prompt": "Generate search keywords for: iPhone 14 Case",
       "response": ["phone accessories", "tablet stand", "device holder"]
     }
   }
   ```

   ❌ **Problem found:** LLM generated generic keywords including "device holder"

3. **Trace the bad candidate through filtering:**

   ```
   GET /api/v1/traces/{id}/events/{filter_event_id}/decisions?item_id=LAPTOP_STAND_123
   ```

   ```json
   {
     "outcome": "accepted",
     "reason_code": "CATEGORY_MATCH",
     "reason_detail": "Both in 'Accessories' category",
     "scores": { "category_similarity": 0.7 }
   }
   ```

   ❌ **Second problem:** Category matching too loose—"Accessories" catches both phone and laptop accessories

4. **Check final ranking:**
   ```json
   {
     "step_name": "llm_rank",
     "annotations": {
       "selected_item": "LAPTOP_STAND_123",
       "llm_reasoning": "Best match for 'device holder' keyword"
     }
   }
   ```
   ❌ **Root cause confirmed:** Bad keywords → wrong candidates → wrong selection

**Fix path:** Improve keyword generation prompt to include product category constraints; tighten category filtering logic.

---

## Queryability

**Cross-pipeline queries work because:**

1. **`step_type` is a first-class field**: When a developer uses `step_type="filter"`, it gets indexed in GSI3 regardless of pipeline. Query:

   ```
   GET /api/v1/query/events?step_type=filter&min_reduction_ratio=0.9
   ```

2. **Standardized metrics**: All filter-like steps record `input_count`, `output_count`, and computed `reduction_ratio`. Developers are _required_ to call `e.set_input()` and `e.set_output()` for this to work.

**Conventions developers must follow:**

- Use consistent `step_type` values across pipelines (e.g., `filter`, `llm`, `transform`)
- Call `set_input()/set_output()` with countable data for reduction ratio
- Use semantic `reason_code` values, not free-text

**Extensibility for unknown use cases:**

The schema is intentionally loose on `annotations`, `metadata`, and `scores`—these are JSON blobs that accept any structure. A log-anomaly-detection pipeline could store `{"anomaly_score": 0.87, "pattern_id": "P123"}` without schema changes.

---

## Performance & Scale

**Problem:** Filter step with 5,000 candidates → 30 survivors. Recording all 5,000 decisions is expensive.

**Solution: SDK-controlled sampling via `SamplingConfig`**

```python
sampling = SamplingConfig({
    "rejected": 0.05,    # 5% of rejections (250 decisions)
    "accepted": 1.0,     # 100% of acceptances (30 decisions)
})
```

**Decision matrix:**

| Capture Mode            | What's Stored             | Decisions Recorded       |
| ----------------------- | ------------------------- | ------------------------ |
| `metrics`               | Counts + timing only      | None                     |
| `sample`                | + 1% deterministic sample | ~50 of 5000              |
| `full`                  | + all decisions           | All 5000                 |
| Custom `SamplingConfig` | Developer-defined rates   | Configurable per outcome |

**Trade-offs:**

- _Storage_: Full capture at scale = ~1KB/decision × 5000 = 5MB/step. Metrics-only = ~500 bytes total.
- _Debug fidelity_: Sample mode may miss the specific item you're debugging. Mitigation: deterministic sampling means same item always sampled/not-sampled.
- _Who decides_: **The developer decides** via `capture` mode. SDK defaults to `metrics` (minimal overhead); developer escalates when debugging.

**Backend optimizations:**

- Batch writes (25 items/batch via DynamoDB BatchWriteItem)
- TTL-based expiration (90 days for traces, 30 days for sampled decisions)
- Sparse GSI5 for item history (only sampled decisions indexed)

---

## Developer Experience

### Minimal Instrumentation (5 minutes)

```python
import xray_sdk as xray
from xray_sdk import XRayStepType, XRayPipelineID

class MySteps(XRayStepType):
    FILTER = "filter"

class MyPipelines(XRayPipelineID):
    SEARCH = "search"

xray.configure(endpoint="http://localhost:8080/api/v1")
xray.register_pipeline(MyPipelines.SEARCH, MySteps)

# Wrap existing pipeline
with xray.trace(MyPipelines.SEARCH) as t:
    with t.event("filter", step_type=MySteps.FILTER) as e:
        e.set_input(candidates)
        results = existing_filter(candidates)  # No changes to business logic
        e.set_output(results)
```

**Result:** Trace with step timing, input/output counts, reduction ratio. Zero changes to business logic.

### Full Instrumentation

```python
with t.event("filter", step_type=MySteps.FILTER, capture="full") as e:
    e.set_input(candidates)
    for item in candidates:
        allowed, reason = check(item)
        e.record_decision(
            item_id=item.id,
            outcome="accepted" if allowed else "rejected",
            reason_code=reason.code,
            reason_detail=f"Score: {item.score}",
            scores={"relevance": item.score},
            item_snapshot={"title": item.title, "price": item.price}
        )
    e.set_output(filtered)
```

### Backend Unavailable

**Graceful degradation:**

```python
xray.configure(
    endpoint="http://api.example.com",
    fallback="local_file",        # Options: "none", "local_file", "memory"
    fallback_path="./xray_backup"
)
```

- `fallback="none"`: Silently drop data (pipeline unaffected)
- `fallback="local_file"`: Write to disk for later ingestion
- `fallback="memory"`: Buffer in memory, retry on reconnect

**Pipeline never blocks on X-Ray failures.** Async sending (default) means `record_decision()` returns immediately; HTTP failures are handled in background thread.

---

## A Sample Root Cause Analysis (RCA) on a Voice Pipeline

From my last week hobby project: Task was to build a Voice Agent for FiveM with Actions: `STT → LLM + Action Handler → TTS`. User asks for money, AI says "Sorry, I can't give!".

The system dynamically evaluates action availability (cooldowns, limits, permissions) and builds LLM prompts. Is this working correctly?

**Debugging Approach:**

1. **LLM Step**: Verify the system prompt contained correct action flags (Check input prompts)
2. **Action Handler Step**: Verify each action was evaluated properly (Check decision outcomes)

### How X-Ray Would Help

**Trace the full conversation:**

```bash
GET /traces/{trace_id}
```

```json
{
  "trace": {
    "pipeline_id": "voice-agent",
    "status": "completed",
    "metadata": { "player_id": "player_123", "session": "abc" }
  },
  "events": [
    { "step_name": "stt", "step_type": "transform", "output_count": 1 },
    { "step_name": "llm_decision", "step_type": "llm", "output_count": 1 },
    {
      "step_name": "action_handler",
      "step_type": "filter",
      "input_count": 5,
      "output_count": 5
    },
    { "step_name": "tts", "step_type": "transform", "output_count": 1 }
  ]
}
```

We see `llm_decision` and `action_handler` both processed. The action_handler shows 5→5 (no filtering), but we need to check the decisions. Let's investigate:

## Step 1: Check LLM Decision

Did the system prompt correctly show available actions? Check the LLM input prompt:

```bash
GET /traces/{trace_id}/events/{llm_event_id}
```

```json
{
  "event_id": "evt_llm_001",
  "step_name": "llm_decision",
  "step_type": "llm",
  "input_sample": [
    "User: Can I have some money?\nSystem: You are a helpful assistant. Available actions: GIVE_MONEY=false (cooldown active), SPAWN_CAR=true, ..."
  ],
  "output_sample": [
    "Assistant: Sorry, I can't give you money right now due to cooldown restrictions."
  ],
  "annotations": {
    "model": "gpt-4",
    "temperature": 0.7,
    "prompt_tokens": 150,
    "completion_tokens": 35
  }
}
```

**Level 1 RCA Complete**: The system prompt correctly showed `GIVE_MONEY=false` due to cooldown. The LLM followed the prompt and said no. The issue is that the action handler should have set `GIVE_MONEY=false` in the prompt, but did it evaluate this correctly?

## Step 2: Check Action Handler Decisions

Now check what the action handler decided for each available action:

```bash
# All decisions from action handler
GET /traces/{trace_id}/events/{action_handler_id}/decisions

# Decisions for GIVE_MONEY specifically
GET /traces/{trace_id}/events/{action_handler_id}/decisions?item_id=GIVE_MONEY
```

```json
{
  "decisions": [
    {
      "item_id": "GIVE_MONEY",
      "outcome": "rejected",
      "reason_code": "COOLDOWN_ACTIVE",
      "reason_detail": "Player received money 12 min ago, cooldown is 30 min",
      "scores": { "minutes_since_last": 12, "cooldown_minutes": 30 }
    },
    {
      "item_id": "SPAWN_CAR",
      "outcome": "accepted",
      "reason_code": "WITHIN_LIMITS",
      "reason_detail": "Player has 0 cars, limit is 3",
      "scores": { "current_count": 0, "max_limit": 3 }
    },
    {
      "item_id": "TELEPORT",
      "outcome": "accepted",
      "reason_code": "PERMISSION_GRANTED",
      "reason_detail": "Basic movement action always allowed"
    },
    {
      "item_id": "HEAL_PLAYER",
      "outcome": "accepted",
      "reason_code": "HEALTH_LOW",
      "reason_detail": "Player health 25%, healing needed",
      "scores": { "current_health": 25, "threshold": 30 }
    },
    {
      "item_id": "GIVE_WEAPON",
      "outcome": "rejected",
      "reason_code": "RESTRICTED_ITEM",
      "reason_detail": "Weapon distribution disabled in safe zones"
    }
  ]
}
```

**Root Cause Found!** The action handler correctly evaluated each action:

- ✅ `GIVE_MONEY`: Rejected (cooldown active → `GIVE_MONEY=false` in prompt)
- ✅ `SPAWN_CAR`: Accepted (within limits → `SPAWN_CAR=true` in prompt)
  ...

The system worked correctly! The action handler properly set `GIVE_MONEY=false` in the LLM prompt based on cooldown rules.

## What Next?

If shipping X-Ray for production:

1. **Authentication/Multi-tenancy:** API keys, org isolation, a dashboard to view traces, events and decisions

2. **Database choice:** While DynamoDB seems to work now, I've read there are many better options like ClickHouse, TimescaleDB, etc. Need to read about them and see how are they better.

3. **Compression:** If we start saving item-snapshots, it could mean very very large dynamoDB items. We'll have to look into compression options.

4. **More query patterns, Natural Language Querying:** We could look into more query patterns considering we have quite some data from each process and expand to natural language querying.

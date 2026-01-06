# X-Ray Python SDK

Reasoning-based observability for multi-step decision pipelines. Capture _why_ decisions were made, not just what happened.

## Installation

(Use this method until its not available on The Python Package Index (PyPI))

```bash
# From the sdk/python directory
pip install -e .

# Or from anywhere, using the full path
pip install -e /path/to/xray-ag/sdk/python
```

## Quick Start

### 1. Define Your Types

```python
import xray_sdk as xray
from xray_sdk import XRayStepType, XRayPipelineID, XRayReasonCode

class MyPipelines(XRayPipelineID):
    SEARCH = "search"

class MySteps(XRayStepType):
    FILTER = "filter"
    LLM = "llm"

class MyReasons(XRayReasonCode):
    PRICE_TOO_HIGH = "PRICE_TOO_HIGH"
    LOW_RATING = "LOW_RATING"
```

### 2. Configure & Register

```python
xray.configure(endpoint="http://localhost:8080/api/v1")
xray.register_pipeline(MyPipelines.SEARCH, MySteps, MyReasons)
```

### 3. Instrument Your Pipeline

```python
with xray.trace(MyPipelines.SEARCH, metadata={"user": "123"}) as t:

    with t.event("filter_products", step_type=MySteps.FILTER, capture="full") as e:
        e.set_input(candidates)

        for item in candidates:
            if item.price > 100:
                e.record_decision(item.id, "rejected",
                    reason_code=MyReasons.PRICE_TOO_HIGH,
                    scores={"price": item.price})
            else:
                filtered.append(item)
                e.record_decision(item.id, "accepted")

        e.set_output(filtered)
```

## Core Concepts

| Concept      | Purpose                                              |
| ------------ | ---------------------------------------------------- |
| **Trace**    | Complete pipeline execution (one per request)        |
| **Event**    | Single step in the pipeline (filter, LLM call, etc.) |
| **Decision** | Per-item outcome with reason code and scores         |

## Capture Modes

```python
# Metrics only (default) - counts and timing
t.event("step", step_type=MySteps.FILTER, capture="metrics")

# Sample mode - 1% of decisions
t.event("step", step_type=MySteps.FILTER, capture="sample")

# Full mode - all decisions
t.event("step", step_type=MySteps.FILTER, capture="full")
```

## Custom Sampling

Control what percentage of each outcome gets stored:

```python
from xray_sdk import SamplingConfig

sampling = SamplingConfig({
    "rejected": 1.0,    # Always store rejections (for debugging)
    "accepted": 0.01,   # 1% of acceptances
})

with xray.trace(MyPipelines.SEARCH, sampling_config=sampling) as t:
    ...
```

## Graceful Degradation

Pipeline never fails if X-Ray backend is down:

```python
xray.configure(
    endpoint="http://api.example.com",
    fallback="local_file",           # "none", "local_file", or "memory"
    fallback_path="./xray_backup",
    async_send=True,                 # Non-blocking (default)
)
```

## Query API

```python
# Get a trace with all events and decisions
trace = xray.get_trace("abc123")

# Query events by step type
results = xray.query(step_type="filter", min_reduction_ratio=0.9)

# Get item history across all traces
history = xray.get_item_history("ASIN-B08N5W")
```

## Configuration Options

| Option                    | Default                         | Description                                |
| ------------------------- | ------------------------------- | ------------------------------------------ |
| `endpoint`                | `http://localhost:8080/api/v1`  | API URL                                    |
| `api_key`                 | `None` (not required right now) | Auth token (or set `XRAY_API_KEY` env var) |
| `async_send`              | `True`                          | Non-blocking sends                         |
| `batch_size`              | `100`                           | Items per batch                            |
| `max_decisions_per_event` | `10000`                         | Limit per event                            |
| `enabled`                 | `True`                          | Set `False` to disable SDK                 |
| `debug`                   | `False`                         | Print debug logs                           |

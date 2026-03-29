# API Documentation

## Event Input/Output Data

### Overview
Events now support capturing and visualizing input and output data samples. When using `SetInput()` and `SetOutput()` in the SDK, the dashboard will display the actual data that was processed.

### Backend Changes

#### Bug Fix: Batch Event Ingestion
Fixed a bug in `/api/v1/events/batch` endpoint where `input_sample` and `output_sample` fields were not being copied from the request to the database model.

**File:** `api/internal/handlers/ingest.go`
**Lines:** 298-300

```go
// Before (missing sample fields):
events[i] = &models.Event{
    EventID:       eventID,
    TraceID:       e.TraceID,
    // ... other fields
    InputCount:    e.InputCount,
    OutputCount:   e.OutputCount,
    // InputSample and OutputSample were missing!
}

// After (fixed):
events[i] = &models.Event{
    EventID:       eventID,
    TraceID:       e.TraceID,
    // ... other fields
    InputCount:    e.InputCount,
    InputSample:   e.InputSample,    // ✅ Added
    OutputCount:   e.OutputCount,
    OutputSample:  e.OutputSample,   // ✅ Added
}
```

### Frontend Changes

#### New Component: EventDataView
**Location:** `dashboard/src/components/trace/EventDataView.tsx`

A reusable component that displays input and output samples in an organized, collapsible format.

**Features:**
- Side-by-side layout for input/output samples
- Expandable/collapsible items
- Data type indicators (object, array, string, number, etc.)
- Preview text for collapsed items
- JSON syntax highlighting for complex objects
- Sample count badges
- Matches existing UI design patterns

#### Integration
**Location:** `dashboard/src/app/traces/[id]/page.tsx`

The EventDataView component is integrated into the trace detail page, appearing when an event is expanded:

1. Input/Output Data (new)
2. Annotations
3. Decisions

### API Response Format

Events now include `input_sample` and `output_sample` arrays:

```json
{
  "event_id": "abc123",
  "step_type": "filter",
  "input_count": 3,
  "output_count": 1,
  "input_sample": [
    {"asin": "B012345678", "price": 1400.0, "category": "electronics"},
    {"asin": "B045678901", "price": 2200.0, "category": "electronics"}
  ],
  "output_sample": [
    {"asin": "B012345678", "price": 1400.0, "category": "electronics"}
  ],
  "annotations": {
    "rejected_count": 2
  }
}
```

### Testing

#### Manual Test
```bash
# Run SDK example
cd sdk/go/examples/basic
go run main.go

# Check API returns samples
TRACE_ID="<trace_id_from_output>"
curl "http://localhost:8080/api/v1/traces/${TRACE_ID}" | jq '.events[0] | {input_sample, output_sample}'
```

#### Dashboard Verification
1. Navigate to http://localhost:3001/traces
2. Click on any trace
3. Expand an event
4. Verify "Input/Output Data" section appears with sample data
5. Click on individual samples to expand/collapse them

### SDK Usage Example

```go
// Example 1: Single object input (now supported!)
filterEvent, _ := trace.StartEvent("filter", xray.EventOptions{
    CaptureMode: xray.CaptureModeFull,
})

// This now works - single map will be wrapped in an array automatically
filterEvent.SetInput(map[string]interface{}{
    "current_speaker": "John",
    "current_text":    "Hello",
    "tool_count":      5,
})

// Array output works as before
messages := []interface{}{
    map[string]interface{}{"role": "system", "content": "You are helpful"},
    map[string]interface{}{"role": "user", "content": "Hi there"},
}
filterEvent.SetOutput(messages, len(messages))
filterEvent.End(nil)

// Example 2: Array input (works as before)
candidates := []map[string]interface{}{
    {"speaker": "John", "text": "Hello", "timestamp": "10:30"},
    {"speaker": "Jane", "text": "Hi there", "timestamp": "10:31"},
}
ingestEvent.SetInput(candidates)
result := process(candidates)
ingestEvent.SetOutput(result)
ingestEvent.End(nil)
```

The dashboard will now display:
- Single objects wrapped in an array with index [0]
- Array items with their respective indices [0], [1], [2], etc.
- All data is expandable/collapsible with proper JSON formatting

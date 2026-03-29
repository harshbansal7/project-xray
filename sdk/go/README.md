# X-Ray Go SDK

Go SDK for reasoning-based observability in multi-step pipelines.

## Installation

```bash
cd sdk/go
go mod tidy
```

Then import in your Go project:

```go
import xray "github.com/xray-sdk/xray-go"
```

## Quick start

```go
xray.Configure(
    xray.WithEndpoint("http://localhost:8080/api/v1"),
    xray.WithAsyncSend(true),
)
defer xray.Shutdown()

xray.RegisterPipeline(
    "competitor-selection",
    []xray.StepType{"filter", "rank", "select"},
    []xray.ReasonCode{"PRICE_TOO_HIGH", "CATEGORY_MISMATCH"},
)

trace, _ := xray.StartTrace("competitor-selection", xray.TraceOptions{
    Metadata: map[string]interface{}{"source": "api"},
})
defer trace.End(nil)

event, _ := trace.StartEvent("filter", xray.EventOptions{CaptureMode: xray.CaptureModeFull})
defer event.End(nil)

event.SetInput(candidates)
for _, item := range candidates {
    reason := "PRICE_TOO_HIGH"
    event.RecordDecision(item.ID, "rejected", xray.DecisionOptions{ReasonCode: &reason})
}
event.SetOutput(filtered)
```

## Functional parity with Python SDK

- Config (`endpoint`, `api_key`, async batching, retries, fallback mode, limits)
- Pipeline registry (step type validation + reason-code registration)
- Trace / event / decision instrumentation
- Capture modes (`metrics`, `sample`, `full`)
- Outcome-based deterministic sampling via `SamplingConfig`
- Async ingestion with batch dedupe and fallback (`none`, `memory`, `local_file`)
- Query helpers (`GetTrace`, `Query`, `QueryAdvanced`, `GetDecisions`, `GetItemHistory`)

## Example

```bash
cd sdk/go
go run ./examples/basic
```

## Notes for AI-assisted integration

- Use `WithTrace` + `WithEvent` helpers if you want guaranteed end/flush behavior in function scopes.
- Use registered `StepType`/`ReasonCode` constants in your project to keep queries consistent.
- Default mode is non-blocking async send; always call `defer xray.Shutdown()` in app startup.
- For debugging correctness issues, start with `CaptureModeFull` on the suspect step only.

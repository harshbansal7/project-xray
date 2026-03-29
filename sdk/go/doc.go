// Package xray provides reasoning-centric observability for multi-step decision pipelines.
//
// The SDK records three levels of data:
//   - Trace: one pipeline execution
//   - Event: one step in the pipeline
//   - Decision: one item-level outcome inside an event
//
// Typical usage:
//
// xray.Configure(
// xray.WithEndpoint("http://localhost:8080/api/v1"),
// xray.WithAsyncSend(true),
// )
// xray.RegisterPipeline("competitor-selection", []xray.StepType{"filter", "rank"}, nil)
//
// trace, _ := xray.StartTrace("competitor-selection", xray.TraceOptions{})
// defer trace.End(nil)
//
// event, _ := trace.StartEvent("filter", xray.EventOptions{CaptureMode: xray.CaptureModeFull})
// defer event.End(nil)
// event.SetInput([]string{"a", "b", "c"})
// event.RecordDecision("a", "accepted", xray.DecisionOptions{})
// event.SetOutput([]string{"a"})
package xray

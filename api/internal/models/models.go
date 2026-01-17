package models

import (
	"time"
)

// StepType represents the type of a pipeline step (string from SDK registry)
type StepType string

// CaptureMode represents how much detail was captured
type CaptureMode string

const (
	CaptureModeMetrics CaptureMode = "metrics"
	CaptureModeSample  CaptureMode = "sample"
	CaptureModeFull    CaptureMode = "full"
)

// Decision outcomes are now flexible strings (no longer enum)

// Trace represents a complete pipeline execution
type Trace struct {
	TraceID    string                 `json:"trace_id"`
	PipelineID string                 `json:"pipeline_id"`
	StartedAt  time.Time              `json:"started_at"`
	EndedAt    *time.Time             `json:"ended_at,omitempty"`
	Status     string                 `json:"status"` // running, completed, failed
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	InputData  interface{}            `json:"input_data,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
}

// EventMetrics is a flexible JSON map for event metrics
// Can contain: duration_ms, input_count, output_count, reduction_ratio, etc.
type EventMetrics map[string]interface{}

// Event represents a single step in a pipeline
type Event struct {
	EventID       string                 `json:"event_id"`
	TraceID       string                 `json:"trace_id"`
	ParentEventID *string                `json:"parent_event_id,omitempty"`
	StepType      StepType               `json:"step_type"`
	CaptureMode   CaptureMode            `json:"capture_mode"`
	InputCount    *int                   `json:"input_count,omitempty"`
	InputSample   []interface{}          `json:"input_sample,omitempty"`
	OutputCount   *int                   `json:"output_count,omitempty"`
	OutputSample  []interface{}          `json:"output_sample,omitempty"`
	Metrics       EventMetrics           `json:"metrics"`
	Annotations   map[string]interface{} `json:"annotations,omitempty"`
	StartedAt     time.Time              `json:"started_at"`
	EndedAt       *time.Time             `json:"ended_at,omitempty"`
}

// Decision represents an individual item decision
type Decision struct {
	DecisionID   string                 `json:"decision_id"`
	EventID      string                 `json:"event_id"`
	TraceID      string                 `json:"trace_id"`
	ItemID       string                 `json:"item_id"`
	Outcome      string                 `json:"outcome"`
	ReasonCode   *string                `json:"reason_code,omitempty"`
	ReasonDetail *string                `json:"reason_detail,omitempty"`
	Scores       map[string]float64     `json:"scores,omitempty"`
	ItemSnapshot map[string]interface{} `json:"item_snapshot,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

// TraceWithEvents is a trace with all its events
type TraceWithEvents struct {
	Trace     Trace                 `json:"trace"`
	Events    []Event               `json:"events"`
	Decisions map[string][]Decision `json:"decisions,omitempty"` // event_id -> decisions
}

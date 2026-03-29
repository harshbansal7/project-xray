package xray

import "time"

// TraceData is the ingest payload for traces.
type TraceData struct {
	TraceID    string                 `json:"trace_id,omitempty"`
	PipelineID string                 `json:"pipeline_id"`
	StartedAt  time.Time              `json:"started_at"`
	EndedAt    *time.Time             `json:"ended_at,omitempty"`
	Status     string                 `json:"status,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	InputData  interface{}            `json:"input_data,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
}

// EventMetrics are computed fields for an event.
type EventMetrics struct {
	InputCount     *int     `json:"input_count,omitempty"`
	OutputCount    *int     `json:"output_count,omitempty"`
	ReductionRatio *float64 `json:"reduction_ratio,omitempty"`
	DurationMS     *float64 `json:"duration_ms,omitempty"`
}

// EventData is the ingest payload for events.
type EventData struct {
	EventID       string                 `json:"event_id,omitempty"`
	TraceID       string                 `json:"trace_id"`
	ParentEventID *string                `json:"parent_event_id,omitempty"`
	StepType      string                 `json:"step_type"`
	CaptureMode   CaptureMode            `json:"capture_mode,omitempty"`
	InputCount    *int                   `json:"input_count,omitempty"`
	InputSample   []interface{}          `json:"input_sample,omitempty"`
	OutputCount   *int                   `json:"output_count,omitempty"`
	OutputSample  []interface{}          `json:"output_sample,omitempty"`
	Metrics       EventMetrics           `json:"metrics,omitempty"`
	Annotations   map[string]interface{} `json:"annotations,omitempty"`
	StartedAt     time.Time              `json:"started_at"`
	EndedAt       *time.Time             `json:"ended_at,omitempty"`
}

// DecisionData is the ingest payload for decisions.
type DecisionData struct {
	DecisionID   string                 `json:"decision_id,omitempty"`
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

// TraceWithEvents is returned by GET /traces/{id}.
type TraceWithEvents struct {
	Trace     map[string]interface{}   `json:"trace"`
	Events    []map[string]interface{} `json:"events"`
	Decisions map[string]interface{}   `json:"decisions,omitempty"`
}

// QueryRequest matches POST /query payload.
type QueryRequest struct {
	PipelineID        *string           `json:"pipeline_id,omitempty"`
	StepType          *string           `json:"step_type,omitempty"`
	MinReductionRatio *float64          `json:"min_reduction_ratio,omitempty"`
	TimeRange         *string           `json:"time_range,omitempty"`
	StartTime         *string           `json:"start_time,omitempty"`
	EndTime           *string           `json:"end_time,omitempty"`
	Tags              []string          `json:"tags,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	Limit             int               `json:"limit,omitempty"`
	Cursor            *string           `json:"cursor,omitempty"`
}

package models

// API request/response types

// CreateTraceRequest is the request body for creating a trace
type CreateTraceRequest struct {
	TraceID    string                 `json:"trace_id"`
	PipelineID string                 `json:"pipeline_id" validate:"required"`
	StartedAt  string                 `json:"started_at" validate:"required"`
	EndedAt    *string                `json:"ended_at,omitempty"`
	Status     string                 `json:"status,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	InputData  interface{}            `json:"input_data,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
}

// UpdateTraceRequest is for updating/completing a trace
type UpdateTraceRequest struct {
	EndedAt *string `json:"ended_at,omitempty"`
	Status  *string `json:"status,omitempty"`
}

// BatchTracesRequest is for batch ingesting traces
type BatchTracesRequest struct {
	Traces []CreateTraceRequest `json:"traces" validate:"required,dive"`
}

// CreateEventRequest is the request body for creating an event
type CreateEventRequest struct {
	EventID       string                 `json:"event_id"`
	TraceID       string                 `json:"trace_id"`
	ParentEventID *string                `json:"parent_event_id,omitempty"`
	StepType      string                 `json:"step_type" validate:"required"`
	CaptureMode   string                 `json:"capture_mode,omitempty"`
	InputCount    *int                   `json:"input_count,omitempty"`
	InputSample   []interface{}          `json:"input_sample,omitempty"`
	OutputCount   *int                   `json:"output_count,omitempty"`
	OutputSample  []interface{}          `json:"output_sample,omitempty"`
	Metrics       *EventMetrics          `json:"metrics,omitempty"`
	Annotations   map[string]interface{} `json:"annotations,omitempty"`
	StartedAt     string                 `json:"started_at" validate:"required"`
	EndedAt       *string                `json:"ended_at,omitempty"`
}

// BatchEventsRequest is for batch ingesting events
type BatchEventsRequest struct {
	Events []CreateEventRequest `json:"events" validate:"required,dive"`
}

// CreateDecisionRequest is the request body for creating a decision
type CreateDecisionRequest struct {
	DecisionID   string                 `json:"decision_id,omitempty"`
	EventID      string                 `json:"event_id"`
	TraceID      string                 `json:"trace_id"`
	ItemID       string                 `json:"item_id" validate:"required"`
	Outcome      string                 `json:"outcome" validate:"required"`
	ReasonCode   *string                `json:"reason_code,omitempty"`
	ReasonDetail *string                `json:"reason_detail,omitempty"`
	Scores       map[string]float64     `json:"scores,omitempty"`
	ItemSnapshot map[string]interface{} `json:"item_snapshot,omitempty"`
	Timestamp    *string                `json:"timestamp,omitempty"`
}

// BatchDecisionsRequest is for batch ingesting decisions
type BatchDecisionsRequest struct {
	Decisions []CreateDecisionRequest `json:"decisions" validate:"required,dive"`
}

// QueryRequest is the request body for querying
type QueryRequest struct {
	PipelineID        *string           `json:"pipeline_id,omitempty"`
	StepType          *string           `json:"step_type,omitempty"`
	MinReductionRatio *float64          `json:"min_reduction_ratio,omitempty"`
	TimeRange         *string           `json:"time_range,omitempty"` // last_24h, last_7d, etc.
	StartTime         *string           `json:"start_time,omitempty"`
	EndTime           *string           `json:"end_time,omitempty"`
	Tags              []string          `json:"tags,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	Limit             int               `json:"limit,omitempty"`
	Cursor            *string           `json:"cursor,omitempty"`
}

// APIResponse is a generic API response
type APIResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// PaginatedResponse is a response with pagination
type PaginatedResponse struct {
	Results    interface{} `json:"results"`
	NextCursor *string     `json:"next_cursor,omitempty"`
	Count      int         `json:"count"`
}

// ErrorResponse is an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

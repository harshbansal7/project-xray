// Package store defines the abstract storage interface for X-Ray data.
// This allows swapping between different database implementations
// (ClickHouse, PostgreSQL, etc.) without changing business logic.
package store

import (
	"context"
	"time"

	"github.com/xray-sdk/xray-api/internal/models"
)

// Store defines the interface for X-Ray data persistence.
// Implementations must be safe for concurrent use.
type Store interface {
	// Trace operations
	CreateTrace(ctx context.Context, trace *models.Trace) error
	UpdateTrace(ctx context.Context, traceID string, updates *TraceUpdates) error
	GetTrace(ctx context.Context, traceID string) (*models.Trace, error)
	GetTraceWithEvents(ctx context.Context, traceID string) (*models.TraceWithEvents, error)
	BatchCreateTraces(ctx context.Context, traces []*models.Trace) error

	// Event operations
	CreateEvent(ctx context.Context, event *models.Event) error
	GetEvent(ctx context.Context, traceID, eventID string) (*models.Event, error)
	GetEventsByTrace(ctx context.Context, traceID string) ([]*models.Event, error)
	BatchCreateEvents(ctx context.Context, events []*models.Event) error

	// Decision operations
	CreateDecision(ctx context.Context, decision *models.Decision) error
	GetDecisionsByEvent(ctx context.Context, eventID string, opts *DecisionQueryOpts) (*DecisionPage, error)
	QueryDecisions(ctx context.Context, opts *DecisionQueryOpts) (*DecisionPage, error)
	BatchCreateDecisions(ctx context.Context, decisions []*models.Decision) error

	// Query operations
	QueryTraces(ctx context.Context, opts *TraceQueryOpts) (*TracePage, error)
	QueryEvents(ctx context.Context, opts *EventQueryOpts) (*EventPage, error)

	// Health check
	Ping(ctx context.Context) error

	// Cleanup
	Close() error
}

// TraceUpdates contains fields that can be updated on a trace
type TraceUpdates struct {
	EndedAt *time.Time
	Status  *string
}

// TraceQueryOpts defines options for querying traces
type TraceQueryOpts struct {
	PipelineID *string
	StartTime  *time.Time
	EndTime    *time.Time
	Status     *string
	Tags       []string
	Metadata   map[string]string
	Limit      int
	Cursor     *string
}

// EventQueryOpts defines options for querying events
type EventQueryOpts struct {
	TraceID           *string
	PipelineID        *string
	StepType          *string
	MinReductionRatio *float64
	StartTime         *time.Time
	EndTime           *time.Time
	Limit             int
	Cursor            *string
}

// DecisionQueryOpts defines options for querying decisions
type DecisionQueryOpts struct {
	TraceID    *string
	PipelineID *string
	StepType   *string
	Outcome    *string
	ReasonCode *string
	ItemID     *string
	Limit      int
	Cursor     *string
}

// TracePage is a paginated list of traces
type TracePage struct {
	Traces     []*models.Trace
	NextCursor *string
}

// EventPage is a paginated list of events
type EventPage struct {
	Events     []*models.Event
	NextCursor *string
}

// DecisionPage is a paginated list of decisions
type DecisionPage struct {
	Decisions  []*models.Decision
	NextCursor *string
}

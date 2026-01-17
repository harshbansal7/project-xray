// Package clickhouse implements the Store interface using ClickHouse.
// This file contains trace-related CRUD operations.
package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/xray-sdk/xray-api/internal/models"
	"github.com/xray-sdk/xray-api/internal/store"
	"github.com/xray-sdk/xray-api/internal/uuidv7"
)

// CreateTrace creates a new trace record
func (s *ClickHouseStore) CreateTrace(ctx context.Context, trace *models.Trace) error {
	if trace.TraceID == "" {
		trace.TraceID = uuidv7.New()
	}

	model := TraceModel{
		TraceID:    trace.TraceID,
		PipelineID: trace.PipelineID,
		StartedAt:  trace.StartedAt,
		EndedAt:    trace.EndedAt,
		Status:     trace.Status,
		Metadata:   JSONMap(trace.Metadata),
		InputData:  JSONAny{Data: trace.InputData},
		Tags:       trace.Tags,
		CreatedAt:  time.Now().UTC(),
	}

	return s.db.WithContext(ctx).Create(&model).Error
}

// UpdateTrace updates an existing trace using ReplacingMergeTree semantics
func (s *ClickHouseStore) UpdateTrace(ctx context.Context, traceID string, updates *store.TraceUpdates) error {
	// For ReplacingMergeTree, we need to insert a new complete row
	// First get the current trace
	var current TraceModel
	err := s.db.WithContext(ctx).Where("trace_id = ?", traceID).First(&current).Error
	if err != nil {
		return fmt.Errorf("failed to find trace for update: %w", err)
	}

	// Apply updates
	if updates.EndedAt != nil {
		current.EndedAt = updates.EndedAt
	}
	if updates.Status != nil {
		current.Status = *updates.Status
	}

	// Insert the updated row (ReplacingMergeTree will handle deduplication)
	err = s.db.WithContext(ctx).Create(&current).Error
	if err != nil {
		return err
	}

	// Force OPTIMIZE to ensure immediate deduplication for development
	// In production, ClickHouse handles this automatically during merges
	s.db.Exec("OPTIMIZE TABLE xray_traces FINAL")

	return nil
}

// GetTrace retrieves a single trace by ID
func (s *ClickHouseStore) GetTrace(ctx context.Context, traceID string) (*models.Trace, error) {
	var model TraceModel
	err := s.db.WithContext(ctx).
		Table("xray_traces FINAL").
		Where("trace_id = ?", traceID).
		Limit(1).
		Find(&model).Error

	if err != nil {
		return nil, err
	}
	if model.TraceID == "" {
		return nil, nil // Or return specific error if prefered
	}

	return model.ToDomain(), nil
}

// GetTraceWithEvents retrieves a trace with all its events and decisions
func (s *ClickHouseStore) GetTraceWithEvents(ctx context.Context, traceID string) (*models.TraceWithEvents, error) {
	// Get trace
	trace, err := s.GetTrace(ctx, traceID)
	if err != nil {
		return nil, err
	}
	if trace == nil {
		return nil, nil
	}

	// Get events
	events, err := s.GetEventsByTrace(ctx, traceID)
	if err != nil {
		return nil, err
	}

	// Get decisions for each event
	decisionsMap := make(map[string][]models.Decision)

	// Optimization: Fetch all decisions for this trace in one query instead of N+1
	var allDecisions []DecisionModel
	if err := s.db.WithContext(ctx).Where("trace_id = ?", traceID).Find(&allDecisions).Error; err != nil {
		return nil, err
	}

	for _, d := range allDecisions {
		decisionsMap[d.EventID] = append(decisionsMap[d.EventID], *d.ToDomain())
	}

	// Convert events slice
	eventList := make([]models.Event, len(events))
	for i, e := range events {
		eventList[i] = *e
	}

	return &models.TraceWithEvents{
		Trace:     *trace,
		Events:    eventList,
		Decisions: decisionsMap,
	}, nil
}

// BatchCreateTraces creates multiple traces in batch
func (s *ClickHouseStore) BatchCreateTraces(ctx context.Context, traces []*models.Trace) error {
	if len(traces) == 0 {
		return nil
	}

	modelSlice := make([]TraceModel, len(traces))
	for i, t := range traces {
		if t.TraceID == "" {
			t.TraceID = uuidv7.New()
		}
		modelSlice[i] = TraceModel{
			TraceID:    t.TraceID,
			PipelineID: t.PipelineID,
			StartedAt:  t.StartedAt,
			EndedAt:    t.EndedAt,
			Status:     t.Status,
			Metadata:   JSONMap(t.Metadata),
			InputData:  JSONAny{Data: t.InputData},
			Tags:       t.Tags,
			CreatedAt:  time.Now().UTC(),
		}
	}

	return s.db.WithContext(ctx).CreateInBatches(modelSlice, 100).Error
}

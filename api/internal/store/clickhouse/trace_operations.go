// Package clickhouse implements the Store interface using ClickHouse.
// This file contains trace-related CRUD operations.
package clickhouse

import (
	"context"
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

// UpdateTrace updates an existing trace
func (s *ClickHouseStore) UpdateTrace(ctx context.Context, traceID string, updates *store.TraceUpdates) error {
	toUpdate := make(map[string]interface{})
	if updates.EndedAt != nil {
		toUpdate["ended_at"] = *updates.EndedAt
	}
	if updates.Status != nil {
		toUpdate["status"] = *updates.Status
	}

	if len(toUpdate) == 0 {
		return nil
	}

	return s.db.WithContext(ctx).Model(&TraceModel{}).
		Where("trace_id = ?", traceID).
		Updates(toUpdate).Error
}

// GetTrace retrieves a single trace by ID
func (s *ClickHouseStore) GetTrace(ctx context.Context, traceID string) (*models.Trace, error) {
	var model TraceModel
	err := s.db.WithContext(ctx).
		// Use FINAL to get the latest state after updates/replacements
		Set("gorm:table_options", "FINAL").
		Where("trace_id = ?", traceID).
		First(&model).Error

	if err != nil {
		return nil, err
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

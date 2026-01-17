// Package clickhouse implements the Store interface using ClickHouse.
// This file contains event-related CRUD operations.
package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/xray-sdk/xray-api/internal/models"
	"github.com/xray-sdk/xray-api/internal/uuidv7"
)

// CreateEvent creates a new event record
func (s *ClickHouseStore) CreateEvent(ctx context.Context, event *models.Event) error {
	if event.EventID == "" {
		event.EventID = uuidv7.New()
	}

	// Get pipeline_id from trace
	trace, err := s.GetTrace(ctx, event.TraceID)
	if err != nil || trace == nil {
		return fmt.Errorf("failed to get trace for event: trace_id=%s", event.TraceID)
	}

	// Compute reduction ratio
	var reductionRatio *float64
	if event.InputCount != nil && event.OutputCount != nil && *event.InputCount > 0 {
		ratio := 1.0 - (float64(*event.OutputCount) / float64(*event.InputCount))
		reductionRatio = &ratio
	}

	model := EventModel{
		EventID:        event.EventID,
		TraceID:        event.TraceID,
		ParentEventID:  event.ParentEventID,
		StepName:       event.StepName,
		StepType:       string(event.StepType),
		CaptureMode:    string(event.CaptureMode),
		InputCount:     event.InputCount,
		InputSample:    JSONArray(event.InputSample),
		OutputCount:    event.OutputCount,
		OutputSample:   JSONArray(event.OutputSample),
		Metrics:        JSONMap(event.Metrics),
		Annotations:    JSONMap(event.Annotations),
		PipelineID:     trace.PipelineID,
		StartedAt:      event.StartedAt,
		EndedAt:        event.EndedAt,
		ReductionRatio: reductionRatio,
		CreatedAt:      time.Now().UTC(),
	}

	return s.db.WithContext(ctx).Create(&model).Error
}

// GetEvent retrieves a single event by trace ID and event ID
func (s *ClickHouseStore) GetEvent(ctx context.Context, traceID, eventID string) (*models.Event, error) {
	var model EventModel
	err := s.db.WithContext(ctx).
		Where("trace_id = ? AND event_id = ?", traceID, eventID).
		First(&model).Error

	if err != nil {
		return nil, err
	}

	return model.ToDomain(), nil
}

// GetEventsByTrace retrieves all events for a trace
func (s *ClickHouseStore) GetEventsByTrace(ctx context.Context, traceID string) ([]*models.Event, error) {
	var eventModels []EventModel
	err := s.db.WithContext(ctx).
		Where("trace_id = ?", traceID).
		Order("started_at ASC").
		Find(&eventModels).Error

	if err != nil {
		return nil, err
	}

	events := make([]*models.Event, len(eventModels))
	for i, m := range eventModels {
		events[i] = m.ToDomain()
	}

	return events, nil
}

// BatchCreateEvents creates multiple events in batch
func (s *ClickHouseStore) BatchCreateEvents(ctx context.Context, events []*models.Event) error {
	if len(events) == 0 {
		return nil
	}

	// Get pipeline IDs for all traces
	traceMap := make(map[string]string)
	uniqueTraceIDs := make(map[string]bool)
	for _, event := range events {
		uniqueTraceIDs[event.TraceID] = true
	}

	// Fetch required traces
	var traces []TraceModel
	traceIDs := make([]string, 0, len(uniqueTraceIDs))
	for id := range uniqueTraceIDs {
		traceIDs = append(traceIDs, id)
	}

	if len(traceIDs) > 0 {
		if err := s.db.WithContext(ctx).Where("trace_id IN ?", traceIDs).Find(&traces).Error; err != nil {
			return fmt.Errorf("failed to fetch traces for batch events: %w", err)
		}
	}
	for _, t := range traces {
		traceMap[t.TraceID] = t.PipelineID
	}

	eventModels := make([]EventModel, len(events))
	for i, event := range events {
		if event.EventID == "" {
			event.EventID = uuidv7.New()
		}

		pipelineID := traceMap[event.TraceID]

		// Compute reduction ratio
		var reductionRatio *float64
		if event.InputCount != nil && event.OutputCount != nil && *event.InputCount > 0 {
			ratio := 1.0 - (float64(*event.OutputCount) / float64(*event.InputCount))
			reductionRatio = &ratio
		}

		eventModels[i] = EventModel{
			EventID:        event.EventID,
			TraceID:        event.TraceID,
			ParentEventID:  event.ParentEventID,
			StepName:       event.StepName,
			StepType:       string(event.StepType),
			CaptureMode:    string(event.CaptureMode),
			InputCount:     event.InputCount,
			InputSample:    JSONArray(event.InputSample),
			OutputCount:    event.OutputCount,
			OutputSample:   JSONArray(event.OutputSample),
			Metrics:        JSONMap(event.Metrics),
			Annotations:    JSONMap(event.Annotations),
			PipelineID:     pipelineID,
			StartedAt:      event.StartedAt,
			EndedAt:        event.EndedAt,
			ReductionRatio: reductionRatio,
			CreatedAt:      time.Now().UTC(),
		}
	}

	return s.db.WithContext(ctx).CreateInBatches(eventModels, 100).Error
}

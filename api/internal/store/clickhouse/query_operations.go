// Package clickhouse implements the Store interface using ClickHouse.
// This file contains query operations for traces and events.
package clickhouse

import (
	"context"
	"fmt"

	"github.com/xray-sdk/xray-api/internal/models"
	"github.com/xray-sdk/xray-api/internal/store"
)

// QueryTraces queries traces with various filters
func (s *ClickHouseStore) QueryTraces(ctx context.Context, opts *store.TraceQueryOpts) (*store.TracePage, error) {
	var traceModels []TraceModel
	query := s.db.WithContext(ctx).Table("xray_traces FINAL")

	// Filter by PipelineID (uses Projection if available)
	if opts.PipelineID != nil {
		query = query.Where("pipeline_id = ?", *opts.PipelineID)
	}

	if opts.Status != nil {
		query = query.Where("status = ?", *opts.Status)
	}
	if opts.StartTime != nil {
		query = query.Where("started_at >= ?", *opts.StartTime)
	}
	if opts.EndTime != nil {
		query = query.Where("started_at <= ?", *opts.EndTime)
	}

	// ClickHouse array has check
	if len(opts.Tags) > 0 {
		for _, tag := range opts.Tags {
			query = query.Where("has(tags, ?)", tag)
		}
	}

	// ClickHouse JSON extraction check
	if len(opts.Metadata) > 0 {
		for key, val := range opts.Metadata {
			query = query.Where("JSONExtractString(metadata, ?) = ?", key, val)
		}
	}

	// Ordering
	// Optimizes for the projection if pipeline_id is present
	if opts.PipelineID != nil {
		query = query.Order("started_at DESC")
	} else {
		query = query.Order("started_at DESC")
	}

	// Pagination
	limit := 100
	if opts.Limit > 0 {
		limit = opts.Limit
	}
	query = query.Limit(limit)

	if err := query.Find(&traceModels).Error; err != nil {
		return nil, fmt.Errorf("query traces: %w", err)
	}

	traces := make([]*models.Trace, len(traceModels))
	for i, tm := range traceModels {
		traces[i] = tm.ToDomain()
	}

	return &store.TracePage{
		Traces:     traces,
		NextCursor: nil, // TODO: Implement cursor-based pagination
	}, nil
}

// QueryEvents queries events with various filters
func (s *ClickHouseStore) QueryEvents(ctx context.Context, opts *store.EventQueryOpts) (*store.EventPage, error) {
	var eventModels []EventModel
	query := s.db.WithContext(ctx).Table("xray_events")

	// Optimize for Primary Key (trace_id, started_at, event_id)
	if opts.TraceID != nil {
		query = query.Where("trace_id = ?", *opts.TraceID)
	}

	if opts.StepType != nil {
		query = query.Where("step_type = ?", *opts.StepType)
	}
	if opts.PipelineID != nil {
		query = query.Where("pipeline_id = ?", *opts.PipelineID)
	}
	if opts.MinReductionRatio != nil {
		query = query.Where("reduction_ratio >= ?", *opts.MinReductionRatio)
	}
	if opts.StartTime != nil {
		query = query.Where("started_at >= ?", *opts.StartTime)
	}
	if opts.EndTime != nil {
		query = query.Where("started_at <= ?", *opts.EndTime)
	}

	query = query.Order("started_at DESC")

	limit := 100
	if opts.Limit > 0 {
		limit = opts.Limit
	}
	query = query.Limit(limit)

	if err := query.Find(&eventModels).Error; err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}

	events := make([]*models.Event, len(eventModels))
	for i, em := range eventModels {
		events[i] = em.ToDomain()
	}

	return &store.EventPage{
		Events:     events,
		NextCursor: nil,
	}, nil
}

// FIXME: Removed s.scanEventFromRows as it's no longer needed with GORM

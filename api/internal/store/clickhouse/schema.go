// Package clickhouse implements the Store interface using ClickHouse.
// This file contains table schema definitions and initialization.
package clickhouse

import (
	"context"
	"fmt"
)

// Schema SQL statements for creating ClickHouse tables
const (
	// TracesTableSQL creates the traces table with ReplacingMergeTree engine
	// Optimized for trace_id lookup AND pipeline_id list views via Projections.
	TracesTableSQL = `
		CREATE TABLE IF NOT EXISTS xray_traces (
			trace_id String,
			pipeline_id LowCardinality(String),
			started_at DateTime64(6),
			ended_at Nullable(DateTime64(6)),
			status LowCardinality(String),
			metadata String CODEC(ZSTD(1)),
			input_data String CODEC(ZSTD(1)),
			tags Array(String),
			created_at DateTime64(6) DEFAULT now64(6),
			
			PROJECTION by_pipeline (
				SELECT * ORDER BY pipeline_id, started_at
			)
		) ENGINE = ReplacingMergeTree(created_at)
		PARTITION BY toYYYYMM(started_at)
		ORDER BY trace_id
		TTL toDate(started_at) + INTERVAL 90 DAY DELETE
		SETTINGS index_granularity = 8192
	`

	// EventsTableSQL creates the events table optimized for step analytics
	EventsTableSQL = `
		CREATE TABLE IF NOT EXISTS xray_events (
			event_id String,
			trace_id String,
			parent_event_id Nullable(String),
			step_type LowCardinality(String),
			capture_mode LowCardinality(String) DEFAULT 'metrics',
			input_count Nullable(Int32),
			input_sample String CODEC(ZSTD(1)),
			output_count Nullable(Int32),
			output_sample String CODEC(ZSTD(1)),
			metrics String CODEC(ZSTD(1)),
			annotations String CODEC(ZSTD(1)),
			pipeline_id LowCardinality(String),
			started_at DateTime64(6),
			ended_at Nullable(DateTime64(6)),
			reduction_ratio Nullable(Float32),
			created_at DateTime64(6) DEFAULT now64(6),

			INDEX idx_step_type step_type TYPE bloom_filter GRANULARITY 4
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(started_at)
		ORDER BY (trace_id, started_at, event_id)
		TTL toDate(started_at) + INTERVAL 90 DAY DELETE
		SETTINGS index_granularity = 8192
	`

	// DecisionsTableSQL creates the decisions table for item-level reasoning
	DecisionsTableSQL = `
		CREATE TABLE IF NOT EXISTS xray_decisions (
			decision_id String,
			event_id String,
			trace_id String,
			item_id String,
			outcome LowCardinality(String),
			reason_code Nullable(String),
			reason_detail Nullable(String),
			scores String CODEC(ZSTD(1)),
			item_snapshot String CODEC(ZSTD(1)),
			timestamp DateTime64(6),
			created_at DateTime64(6) DEFAULT now64(6),
			
			INDEX idx_item_id item_id TYPE bloom_filter GRANULARITY 4
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(timestamp)
		ORDER BY (event_id, decision_id)
		TTL toDate(timestamp) + INTERVAL 30 DAY DELETE
		SETTINGS index_granularity = 8192
	`
)

// InitSchema initializes all required tables in ClickHouse
func (s *ClickHouseStore) InitSchema(ctx context.Context) error {
	schemas := []struct {
		name string
		sql  string
	}{
		{"traces", TracesTableSQL},
		{"events", EventsTableSQL},
		{"decisions", DecisionsTableSQL},
	}

	for _, schema := range schemas {
		if err := s.db.Exec(schema.sql).Error; err != nil {
			return fmt.Errorf("failed to create %s table: %w", schema.name, err)
		}
	}

	return nil
}

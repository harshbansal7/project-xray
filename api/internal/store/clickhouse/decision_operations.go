// Package clickhouse implements the Store interface using ClickHouse.
// This file contains decision-related CRUD operations.
package clickhouse

import (
	"context"
	"time"

	"github.com/xray-sdk/xray-api/internal/models"
	"github.com/xray-sdk/xray-api/internal/store"
	"github.com/xray-sdk/xray-api/internal/uuidv7"
)

// CreateDecision creates a new decision record
func (s *ClickHouseStore) CreateDecision(ctx context.Context, decision *models.Decision) error {
	if decision.DecisionID == "" {
		decision.DecisionID = uuidv7.New()
	}

	model := DecisionModel{
		DecisionID:   decision.DecisionID,
		EventID:      decision.EventID,
		TraceID:      decision.TraceID,
		ItemID:       decision.ItemID,
		Outcome:      decision.Outcome,
		ReasonCode:   decision.ReasonCode,
		ReasonDetail: decision.ReasonDetail,
		Scores:       JSONScores(decision.Scores),
		ItemSnapshot: JSONMap(decision.ItemSnapshot),
		Timestamp:    decision.Timestamp,
		CreatedAt:    time.Now().UTC(),
	}

	return s.db.WithContext(ctx).Create(&model).Error
}

// GetDecisionsByEvent retrieves all decisions for an event
func (s *ClickHouseStore) GetDecisionsByEvent(ctx context.Context, eventID string, opts *store.DecisionQueryOpts) (*store.DecisionPage, error) {
	var decisionModels []DecisionModel
	query := s.db.WithContext(ctx).Table("xray_decisions").Where("event_id = ?", eventID)

	// Apply filtering if provided in options (though typically this method is just by event_id)
	if opts != nil {
		if opts.TraceID != nil {
			query = query.Where("trace_id = ?", *opts.TraceID)
		}
		// Add other filters as needed
	}

	query = query.Order("timestamp ASC")

	if err := query.Find(&decisionModels).Error; err != nil {
		return nil, err
	}

	decisions := make([]*models.Decision, len(decisionModels))
	for i, dm := range decisionModels {
		decisions[i] = dm.ToDomain()
	}

	return &store.DecisionPage{
		Decisions:  decisions,
		NextCursor: nil,
	}, nil
}

// BatchCreateDecisions creates multiple decisions in batch
func (s *ClickHouseStore) BatchCreateDecisions(ctx context.Context, decisions []*models.Decision) error {
	if len(decisions) == 0 {
		return nil
	}

	models := make([]DecisionModel, len(decisions))
	for i, d := range decisions {
		if d.DecisionID == "" {
			d.DecisionID = uuidv7.New()
		}
		models[i] = DecisionModel{
			DecisionID:   d.DecisionID,
			EventID:      d.EventID,
			TraceID:      d.TraceID,
			ItemID:       d.ItemID,
			Outcome:      d.Outcome,
			ReasonCode:   d.ReasonCode,
			ReasonDetail: d.ReasonDetail,
			Scores:       JSONScores(d.Scores),
			ItemSnapshot: JSONMap(d.ItemSnapshot),
			Timestamp:    d.Timestamp,
			CreatedAt:    time.Now().UTC(),
		}
	}

	return s.db.WithContext(ctx).CreateInBatches(models, 100).Error
}

// QueryDecisions allows flexible querying of decisions
// This requires joining with events for filters like pipeline_id or step_type
func (s *ClickHouseStore) QueryDecisions(ctx context.Context, opts *store.DecisionQueryOpts) (*store.DecisionPage, error) {
	var decisionModels []DecisionModel
	query := s.db.WithContext(ctx).Table("xray_decisions").
		Select("xray_decisions.*").
		Joins("JOIN xray_events ON xray_decisions.event_id = xray_events.event_id")

	if opts.PipelineID != nil {
		query = query.Where("xray_events.pipeline_id = ?", *opts.PipelineID)
	}
	if opts.StepType != nil {
		query = query.Where("xray_events.step_type = ?", *opts.StepType)
	}

	if opts.TraceID != nil {
		query = query.Where("xray_decisions.trace_id = ?", *opts.TraceID)
	}
	if opts.Outcome != nil {
		query = query.Where("xray_decisions.outcome = ?", *opts.Outcome)
	}
	if opts.ReasonCode != nil {
		query = query.Where("xray_decisions.reason_code = ?", *opts.ReasonCode)
	}
	if opts.ItemID != nil {
		query = query.Where("xray_decisions.item_id = ?", *opts.ItemID)
	}

	query = query.Order("xray_decisions.timestamp DESC")

	limit := 100
	if opts.Limit > 0 {
		limit = opts.Limit
	}
	query = query.Limit(limit)

	if err := query.Find(&decisionModels).Error; err != nil {
		return nil, err
	}

	decisions := make([]*models.Decision, len(decisionModels))
	for i, dm := range decisionModels {
		decisions[i] = dm.ToDomain()
	}

	return &store.DecisionPage{
		Decisions:  decisions,
		NextCursor: nil,
	}, nil
}

// Package dynamodb implements the Store interface using AWS DynamoDB.
// This file contains query operations for traces and events.
package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/xray-sdk/xray-api/internal/models"
	"github.com/xray-sdk/xray-api/internal/store"
)

// QueryTraces queries traces with various filters
func (s *DynamoDBStore) QueryTraces(ctx context.Context, opts *store.TraceQueryOpts) (*store.TracePage, error) {
	if opts.PipelineID == nil {
		// If no pipeline specified, return empty (prevent full table scan)
		return &store.TracePage{Traces: []*models.Trace{}}, nil
	}

	// Query GSI2 for traces by pipeline
	// SK is just trace_id (UUIDv7), which is naturally time-sorted
	keyCondition := "GSI2PK = :pipeline"
	exprValues := map[string]types.AttributeValue{
		":pipeline": &types.AttributeValueMemberS{Value: GSI2PK(*opts.PipelineID)},
	}

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(s.tableName),
		IndexName:                 aws.String(GSI2Name),
		KeyConditionExpression:    aws.String(keyCondition),
		ExpressionAttributeValues: exprValues,
		ScanIndexForward:          aws.Bool(false), // Newest first (UUIDv7 ordering)
	}

	if opts.Limit > 0 {
		input.Limit = aws.Int32(int32(opts.Limit))
	}

	// Build filter expression for optional filters
	var filterExprs []string
	exprNames := map[string]string{}

	// Status filter
	if opts.Status != nil {
		filterExprs = append(filterExprs, "#status = :statusVal")
		exprNames["#status"] = "status"
		exprValues[":statusVal"] = &types.AttributeValueMemberS{Value: *opts.Status}
	}

	// Time range filter (now using filter expression since we removed date from SK)
	if opts.StartTime != nil {
		filterExprs = append(filterExprs, "started_at >= :startTime")
		exprValues[":startTime"] = &types.AttributeValueMemberS{Value: opts.StartTime.Format(time.RFC3339)}
	}
	if opts.EndTime != nil {
		filterExprs = append(filterExprs, "started_at <= :endTime")
		exprValues[":endTime"] = &types.AttributeValueMemberS{Value: opts.EndTime.Format(time.RFC3339)}
	}

	if len(filterExprs) > 0 {
		filterExpr := filterExprs[0]
		for i := 1; i < len(filterExprs); i++ {
			filterExpr += " AND " + filterExprs[i]
		}
		input.FilterExpression = aws.String(filterExpr)
	}

	if len(exprNames) > 0 {
		input.ExpressionAttributeNames = exprNames
	}

	result, err := s.client.Query(ctx, input)
	if err != nil {
		return nil, err
	}

	traces := make([]*models.Trace, 0, len(result.Items))
	for _, item := range result.Items {
		trace := &models.Trace{}
		if err := attributevalue.UnmarshalMap(item, trace); err != nil {
			continue
		}
		traces = append(traces, trace)
	}

	var nextCursor *string
	if result.LastEvaluatedKey != nil {
		nextCursor = aws.String("cursor") // Simplified
	}

	return &store.TracePage{
		Traces:     traces,
		NextCursor: nextCursor,
	}, nil
}

// QueryEvents queries events with various filters
func (s *DynamoDBStore) QueryEvents(ctx context.Context, opts *store.EventQueryOpts) (*store.EventPage, error) {
	if opts.StepType == nil {
		return &store.EventPage{Events: []*models.Event{}}, nil
	}

	// Use GSI3 for step-type queries
	var gsi3PK string
	if opts.PipelineID != nil {
		// Pipeline-specific query
		gsi3PK = GSI3PKWithPipeline(*opts.StepType, *opts.PipelineID)
	} else {
		// Global step-type query
		gsi3PK = GSI3PK(*opts.StepType)
	}

	keyCondition := "GSI3PK = :stepKey"
	exprValues := map[string]types.AttributeValue{
		":stepKey": &types.AttributeValueMemberS{Value: gsi3PK},
	}

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(s.tableName),
		IndexName:                 aws.String(GSI3Name),
		KeyConditionExpression:    aws.String(keyCondition),
		ExpressionAttributeValues: exprValues,
		ScanIndexForward:          aws.Bool(false), // Newest first (UUIDv7 ordering)
	}

	if opts.Limit > 0 {
		input.Limit = aws.Int32(int32(opts.Limit))
	}

	// Build filter expression for optional filters
	var filterExprs []string

	// Time range filter
	if opts.StartTime != nil {
		filterExprs = append(filterExprs, "started_at >= :startTime")
		exprValues[":startTime"] = &types.AttributeValueMemberS{Value: opts.StartTime.Format(time.RFC3339)}
	}
	if opts.EndTime != nil {
		filterExprs = append(filterExprs, "started_at <= :endTime")
		exprValues[":endTime"] = &types.AttributeValueMemberS{Value: opts.EndTime.Format(time.RFC3339)}
	}

	if len(filterExprs) > 0 {
		filterExpr := filterExprs[0]
		for i := 1; i < len(filterExprs); i++ {
			filterExpr += " AND " + filterExprs[i]
		}
		input.FilterExpression = aws.String(filterExpr)
	}

	result, err := s.client.Query(ctx, input)
	if err != nil {
		return nil, err
	}

	events := make([]*models.Event, 0, len(result.Items))
	for _, item := range result.Items {
		event := &models.Event{}
		if err := attributevalue.UnmarshalMap(item, event); err != nil {
			continue
		}

		// In-memory filtering for MinReductionRatio (no longer in SK)
		if opts.MinReductionRatio != nil {
			if event.InputCount != nil && event.OutputCount != nil && *event.InputCount > 0 {
				ratio := 1.0 - (float64(*event.OutputCount) / float64(*event.InputCount))
				if ratio < *opts.MinReductionRatio {
					continue // Skip events below the threshold
				}
			} else {
				continue // Skip events without counts when filtering by ratio
			}
		}

		events = append(events, event)
	}

	var nextCursor *string
	if result.LastEvaluatedKey != nil {
		nextCursor = aws.String("cursor") // Simplified
	}

	return &store.EventPage{
		Events:     events,
		NextCursor: nextCursor,
	}, nil
}

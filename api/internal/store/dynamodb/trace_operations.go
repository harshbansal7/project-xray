// Package dynamodb implements the Store interface using AWS DynamoDB.
// This file contains trace-related CRUD operations.
package dynamodb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/xray-sdk/xray-api/internal/models"
	"github.com/xray-sdk/xray-api/internal/store"
	"github.com/xray-sdk/xray-api/internal/uuidv7"
)

// CreateTrace creates a new trace record
func (s *DynamoDBStore) CreateTrace(ctx context.Context, trace *models.Trace) error {
	if trace.TraceID == "" {
		trace.TraceID = uuidv7.New()
	}

	item, err := attributevalue.MarshalMap(trace)
	if err != nil {
		return fmt.Errorf("marshal trace: %w", err)
	}

	deleteGSIKeys(item)

	// Primary key: TRACE#<trace_id> / TRACE#META
	setStringAttr(item, "PK", TracePK(trace.TraceID))
	setStringAttr(item, "SK", SKTrace)
	setStringAttr(item, "entity_type", "TRACE")

	// GSI2: Pipeline queries - PK: GSI2_PIPE#<pipeline_id>, SK: <trace_id>
	setStringAttr(item, "GSI2PK", GSI2PK(trace.PipelineID))
	setStringAttr(item, "GSI2SK", trace.TraceID)

	// Set TTL (90 days from creation)
	if trace.TTL == nil {
		ttl := trace.StartedAt.AddDate(0, 0, TraceTTLDays).Unix()
		setTTLAttr(item, ttl)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	return err
}

// UpdateTrace updates an existing trace
func (s *DynamoDBStore) UpdateTrace(ctx context.Context, traceID string, updates *store.TraceUpdates) error {
	updateExpr := "SET "
	exprNames := map[string]string{}
	exprValues := map[string]types.AttributeValue{}

	if updates.EndedAt != nil {
		updateExpr += "#endedAt = :endedAt, "
		exprNames["#endedAt"] = "ended_at"
		exprValues[":endedAt"] = &types.AttributeValueMemberS{Value: updates.EndedAt.Format("2006-01-02T15:04:05.000Z07:00")}
	}

	if updates.Status != nil {
		updateExpr += "#status = :status, "
		exprNames["#status"] = "status"
		exprValues[":status"] = &types.AttributeValueMemberS{Value: *updates.Status}
	}

	if len(exprValues) == 0 {
		return nil
	}

	updateExpr = updateExpr[:len(updateExpr)-2]

	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: TracePK(traceID)},
			"SK": &types.AttributeValueMemberS{Value: SKTrace},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: exprValues,
	})
	return err
}

// GetTrace retrieves a single trace by ID
func (s *DynamoDBStore) GetTrace(ctx context.Context, traceID string) (*models.Trace, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: TracePK(traceID)},
			"SK": &types.AttributeValueMemberS{Value: SKTrace},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, nil
	}

	var trace models.Trace
	if err := attributevalue.UnmarshalMap(result.Item, &trace); err != nil {
		return nil, fmt.Errorf("unmarshal trace: %w", err)
	}

	return &trace, nil
}

// GetTraceWithEvents retrieves a trace with all its events and decisions
func (s *DynamoDBStore) GetTraceWithEvents(ctx context.Context, traceID string) (*models.TraceWithEvents, error) {
	// First, get the trace using GetItem (direct lookup)
	trace, err := s.GetTrace(ctx, traceID)
	if err != nil {
		return nil, err
	}
	if trace == nil {
		return nil, nil
	}

	// Get events for this trace
	events, err := s.GetEventsByTrace(ctx, traceID)
	if err != nil {
		return nil, err
	}

	// Get decisions for each event
	decisionsMap := make(map[string][]models.Decision)
	for _, event := range events {
		page, err := s.GetDecisionsByEvent(ctx, event.EventID, nil)
		if err == nil && page != nil {
			// Convert []*Decision to []Decision
			decisions := make([]models.Decision, len(page.Decisions))
			for i, d := range page.Decisions {
				decisions[i] = *d
			}
			decisionsMap[event.EventID] = decisions
		}
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
func (s *DynamoDBStore) BatchCreateTraces(ctx context.Context, traces []*models.Trace) error {
	const batchSize = 25

	for i := 0; i < len(traces); i += batchSize {
		end := i + batchSize
		if end > len(traces) {
			end = len(traces)
		}
		batch := traces[i:end]

		writeRequests := make([]types.WriteRequest, len(batch))
		for j, trace := range batch {
			if trace.TraceID == "" {
				trace.TraceID = uuidv7.New()
			}

			item, _ := attributevalue.MarshalMap(trace)
			deleteGSIKeys(item)

			// Primary key
			setStringAttr(item, "PK", TracePK(trace.TraceID))
			setStringAttr(item, "SK", SKTrace)
			setStringAttr(item, "entity_type", "TRACE")

			// GSI2: Pipeline queries
			setStringAttr(item, "GSI2PK", GSI2PK(trace.PipelineID))
			setStringAttr(item, "GSI2SK", trace.TraceID)

			if trace.TTL == nil {
				ttl := trace.StartedAt.AddDate(0, 0, TraceTTLDays).Unix()
				setTTLAttr(item, ttl)
			}

			writeRequests[j] = types.WriteRequest{
				PutRequest: &types.PutRequest{Item: item},
			}
		}

		_, err := s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				s.tableName: writeRequests,
			},
		})
		if err != nil {
			return fmt.Errorf("batch write traces: %w", err)
		}
	}
	return nil
}

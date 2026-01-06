// Package dynamodb implements the Store interface using AWS DynamoDB.
// This file contains event-related CRUD operations.
package dynamodb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/xray-sdk/xray-api/internal/models"
	"github.com/xray-sdk/xray-api/internal/uuidv7"
)

// CreateEvent creates a new event record
func (s *DynamoDBStore) CreateEvent(ctx context.Context, event *models.Event) error {
	if event.EventID == "" {
		event.EventID = uuidv7.New()
	}

	item, err := attributevalue.MarshalMap(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	deleteGSIKeys(item)

	// Primary key: TRACE#<trace_id> / EVENT#<event_id>
	setStringAttr(item, "PK", TracePK(event.TraceID))
	setStringAttr(item, "SK", EventSK(event.EventID))
	setStringAttr(item, "entity_type", "EVENT")

	// Get pipeline_id from trace (needed for GSI3)
	trace, err := s.GetTrace(ctx, event.TraceID)
	if err != nil || trace == nil {
		return fmt.Errorf("failed to get trace for event: %w", err)
	}

	// Store pipeline_id for filtering
	setStringAttr(item, "pipeline_id", trace.PipelineID)

	// GSI3: Global step-type analytics - PK: GSI3_STEP#<step_type>, SK: <event_id>
	setStringAttr(item, "GSI3PK", GSI3PK(string(event.StepType)))
	setStringAttr(item, "GSI3SK", event.EventID)

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return err
	}

	// Write second item for pipeline-specific analytics
	item2, _ := attributevalue.MarshalMap(event)
	deleteGSIKeys(item2)

	// Use EVENT_META prefix to differentiate from main event
	setStringAttr(item2, "PK", TracePK(event.TraceID))
	setStringAttr(item2, "SK", "EVENT_META#"+event.EventID)
	setStringAttr(item2, "entity_type", "EVENT_ANALYTICS")
	setStringAttr(item2, "pipeline_id", trace.PipelineID)

	// GSI3: Pipeline-specific analytics
	setStringAttr(item2, "GSI3PK", GSI3PKWithPipeline(string(event.StepType), trace.PipelineID))
	setStringAttr(item2, "GSI3SK", event.EventID)

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item2,
	})

	return err
}

// GetEvent retrieves a single event by trace ID and event ID
func (s *DynamoDBStore) GetEvent(ctx context.Context, traceID, eventID string) (*models.Event, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: TracePK(traceID)},
			"SK": &types.AttributeValueMemberS{Value: EventSK(eventID)},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, nil
	}

	var event models.Event
	if err := attributevalue.UnmarshalMap(result.Item, &event); err != nil {
		return nil, fmt.Errorf("unmarshal event: %w", err)
	}

	return &event, nil
}

// GetEventsByTrace retrieves all events for a trace
func (s *DynamoDBStore) GetEventsByTrace(ctx context.Context, traceID string) ([]*models.Event, error) {
	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk_prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":        &types.AttributeValueMemberS{Value: TracePK(traceID)},
			":sk_prefix": &types.AttributeValueMemberS{Value: PrefixEvent},
		},
	})
	if err != nil {
		return nil, err
	}

	events := make([]*models.Event, 0, len(result.Items))
	for _, item := range result.Items {
		event := &models.Event{}
		if err := attributevalue.UnmarshalMap(item, event); err != nil {
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// BatchCreateEvents creates multiple events in batch
func (s *DynamoDBStore) BatchCreateEvents(ctx context.Context, events []*models.Event) error {
	// Get unique trace IDs to fetch pipeline_ids
	traceMap := make(map[string]string) // trace_id -> pipeline_id
	uniqueTraceIDs := make(map[string]bool)
	for _, event := range events {
		uniqueTraceIDs[event.TraceID] = true
	}

	// Fetch all traces
	for traceID := range uniqueTraceIDs {
		trace, err := s.GetTrace(ctx, traceID)
		if err == nil && trace != nil {
			traceMap[traceID] = trace.PipelineID
		}
	}

	const batchSize = 25
	allRequests := make([]types.WriteRequest, 0, len(events)*2) // 2x for dual write

	for _, event := range events {
		if event.EventID == "" {
			event.EventID = uuidv7.New()
		}

		pipelineID := traceMap[event.TraceID]

		// Item 1: Main event record
		item1, _ := attributevalue.MarshalMap(event)
		deleteGSIKeys(item1)

		setStringAttr(item1, "PK", TracePK(event.TraceID))
		setStringAttr(item1, "SK", EventSK(event.EventID))
		setStringAttr(item1, "entity_type", "EVENT")
		setStringAttr(item1, "pipeline_id", pipelineID)
		setStringAttr(item1, "GSI3PK", GSI3PK(string(event.StepType)))
		setStringAttr(item1, "GSI3SK", event.EventID)

		allRequests = append(allRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{Item: item1},
		})

		// Item 2: Pipeline-specific analytics
		item2, _ := attributevalue.MarshalMap(event)
		deleteGSIKeys(item2)

		setStringAttr(item2, "PK", TracePK(event.TraceID))
		setStringAttr(item2, "SK", "EVENT_META#"+event.EventID)
		setStringAttr(item2, "entity_type", "EVENT_ANALYTICS")
		setStringAttr(item2, "pipeline_id", pipelineID)
		setStringAttr(item2, "GSI3PK", GSI3PKWithPipeline(string(event.StepType), pipelineID))
		setStringAttr(item2, "GSI3SK", event.EventID)

		allRequests = append(allRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{Item: item2},
		})
	}

	// Batch write all requests
	for i := 0; i < len(allRequests); i += batchSize {
		end := i + batchSize
		if end > len(allRequests) {
			end = len(allRequests)
		}

		_, err := s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				s.tableName: allRequests[i:end],
			},
		})
		if err != nil {
			return fmt.Errorf("batch write events: %w", err)
		}
	}

	return nil
}

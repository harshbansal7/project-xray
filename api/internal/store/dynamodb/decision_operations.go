// Package dynamodb implements the Store interface using AWS DynamoDB.
// This file contains decision-related CRUD operations.
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

// CreateDecision creates a new decision record
func (s *DynamoDBStore) CreateDecision(ctx context.Context, decision *models.Decision) error {
	if decision.DecisionID == "" {
		decision.DecisionID = uuidv7.New()
	}

	item, err := attributevalue.MarshalMap(decision)
	if err != nil {
		return fmt.Errorf("marshal decision: %w", err)
	}

	deleteGSIKeys(item)

	// Primary key: TRACE#<trace_id> / DEC#<event_id>#<decision_id>
	setStringAttr(item, "PK", TracePK(decision.TraceID))
	setStringAttr(item, "SK", DecisionSK(decision.EventID, decision.DecisionID))
	setStringAttr(item, "entity_type", "DECISION")

	// GSI4: Event decisions - PK: GSI4_EVT#<event_id>, SK: <outcome>#<decision_id>
	setStringAttr(item, "GSI4PK", GSI4PK(decision.EventID))
	setStringAttr(item, "GSI4SK", GSI4SK(string(decision.Outcome), decision.DecisionID))

	// GSI5: Item history (SPARSE - index all received decisions)
	// Python SDK handles sampling via SamplingConfig, so we index everything we receive
	// This provides generic support for any outcome type with configurable sampling
	shouldIndex := true
	if shouldIndex {
		setStringAttr(item, "GSI5PK", GSI5PK(decision.ItemID))
		setStringAttr(item, "GSI5SK", decision.DecisionID)

		// Set TTL for all indexed items (90 days - same as traces)
		// Since Python SDK handles sampling, we keep all received decisions
		if decision.TTL == nil {
			ttl := decision.Timestamp.AddDate(0, 0, TraceTTLDays).Unix()
			setTTLAttr(item, ttl)
		}
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	return err
}

// GetDecisionsByEvent retrieves decisions for an event with optional filtering
func (s *DynamoDBStore) GetDecisionsByEvent(ctx context.Context, eventID string, opts *store.DecisionQueryOpts) (*store.DecisionPage, error) {
	keyCondition := "GSI4PK = :eventId"
	exprValues := map[string]types.AttributeValue{
		":eventId": &types.AttributeValueMemberS{Value: GSI4PK(eventID)},
	}
	exprNames := map[string]string{}
	filterExprs := []string{}

	// If outcome is specified, use it in the sort key condition (no filter expression needed!)
	if opts != nil && opts.Outcome != nil {
		keyCondition += " AND begins_with(GSI4SK, :outcome)"
		exprValues[":outcome"] = &types.AttributeValueMemberS{Value: *opts.Outcome + "#"}
	}

	// Additional filters using FilterExpression
	if opts != nil && opts.ReasonCode != nil {
		filterExprs = append(filterExprs, "#reasonCode = :reasonCode")
		exprNames["#reasonCode"] = "reason_code"
		exprValues[":reasonCode"] = &types.AttributeValueMemberS{Value: *opts.ReasonCode}
	}

	if opts != nil && opts.ItemID != nil {
		filterExprs = append(filterExprs, "#itemId = :itemId")
		exprNames["#itemId"] = "item_id"
		exprValues[":itemId"] = &types.AttributeValueMemberS{Value: *opts.ItemID}
	}

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(s.tableName),
		IndexName:                 aws.String(GSI4Name),
		KeyConditionExpression:    aws.String(keyCondition),
		ExpressionAttributeValues: exprValues,
	}

	if opts != nil && opts.Limit > 0 {
		input.Limit = aws.Int32(int32(opts.Limit))
	}

	// Add filter expression if we have filters
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

	decisions := make([]*models.Decision, 0, len(result.Items))
	for _, item := range result.Items {
		decision := &models.Decision{}
		if err := attributevalue.UnmarshalMap(item, decision); err != nil {
			continue
		}
		decisions = append(decisions, decision)
	}

	var nextCursor *string
	if result.LastEvaluatedKey != nil {
		nextCursor = aws.String("cursor") // Simplified cursor
	}

	return &store.DecisionPage{
		Decisions:  decisions,
		NextCursor: nextCursor,
	}, nil
}

// GetDecisionsByItem retrieves decision history for an item
func (s *DynamoDBStore) GetDecisionsByItem(ctx context.Context, itemID string, limit int) ([]*models.Decision, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		IndexName:              aws.String(GSI5Name),
		KeyConditionExpression: aws.String("GSI5PK = :itemId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":itemId": &types.AttributeValueMemberS{Value: GSI5PK(itemID)},
		},
		ScanIndexForward: aws.Bool(false), // Newest first (UUIDv7 is time-sortable)
	}

	if limit > 0 {
		input.Limit = aws.Int32(int32(limit))
	}

	result, err := s.client.Query(ctx, input)
	if err != nil {
		return nil, err
	}

	decisions := make([]*models.Decision, 0, len(result.Items))
	for _, item := range result.Items {
		decision := &models.Decision{}
		if err := attributevalue.UnmarshalMap(item, decision); err != nil {
			continue
		}
		decisions = append(decisions, decision)
	}

	return decisions, nil
}

// BatchCreateDecisions creates multiple decisions in batch
func (s *DynamoDBStore) BatchCreateDecisions(ctx context.Context, decisions []*models.Decision) error {
	const batchSize = 25

	for i := 0; i < len(decisions); i += batchSize {
		end := i + batchSize
		if end > len(decisions) {
			end = len(decisions)
		}
		batch := decisions[i:end]

		writeRequests := make([]types.WriteRequest, len(batch))
		for j, decision := range batch {
			if decision.DecisionID == "" {
				decision.DecisionID = uuidv7.New()
			}

			item, _ := attributevalue.MarshalMap(decision)
			deleteGSIKeys(item)

			// Primary key
			setStringAttr(item, "PK", TracePK(decision.TraceID))
			setStringAttr(item, "SK", DecisionSK(decision.EventID, decision.DecisionID))
			setStringAttr(item, "entity_type", "DECISION")

			// GSI4: Event decisions
			setStringAttr(item, "GSI4PK", GSI4PK(decision.EventID))
			setStringAttr(item, "GSI4SK", GSI4SK(string(decision.Outcome), decision.DecisionID))

			// Sparse GSI5: Item history (index all received decisions)
			// Python SDK handles sampling, so we index everything we receive
			setStringAttr(item, "GSI5PK", GSI5PK(decision.ItemID))
			setStringAttr(item, "GSI5SK", decision.DecisionID)

			// Set TTL for all indexed items (90 days - same as traces)
			if decision.TTL == nil {
				ttl := decision.Timestamp.AddDate(0, 0, TraceTTLDays).Unix()
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
			return fmt.Errorf("batch write decisions: %w", err)
		}
	}
	return nil
}

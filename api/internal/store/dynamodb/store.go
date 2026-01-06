// Package dynamodb implements the Store interface using AWS DynamoDB.
// Production-grade schema with hierarchical keys, optimized GSIs, and efficient access patterns.
package dynamodb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

const (
	TableName = "xray_data"

	// GSI Names
	GSI2Name = "pipeline-time-index"       // Traces by pipeline + time
	GSI3Name = "step-type-analytics-index" // Cross-pipeline event analytics
	GSI4Name = "event-decisions-index"     // Decisions by event with outcome filtering
	GSI5Name = "item-history-index"        // Item decision history (SPARSE)
)

const (
	// TTL durations
	TraceTTLDays    = 90 // Traces expire after 90 days
	DecisionTTLDays = 30 // Sampled decisions expire after 30 days
)

// DynamoDBStore implements store.Store using DynamoDB
type DynamoDBStore struct {
	client    *dynamodb.Client
	tableName string
}

// Config holds DynamoDB store configuration
type Config struct {
	TableName string
	Endpoint  string
	Region    string
}

// New creates a new DynamoDB store
func New(ctx context.Context, cfg Config) (*DynamoDBStore, error) {
	var opts []func(*config.LoadOptions) error

	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}
	opts = append(opts, config.WithRegion(region))

	if cfg.Endpoint != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("local", "local", ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	clientOpts := []func(*dynamodb.Options){}
	if cfg.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	client := dynamodb.NewFromConfig(awsCfg, clientOpts...)
	tableName := cfg.TableName
	if tableName == "" {
		tableName = TableName
	}

	return &DynamoDBStore{client: client, tableName: tableName}, nil
}

// Ping checks if the DynamoDB table is accessible
func (s *DynamoDBStore) Ping(ctx context.Context) error {
	_, err := s.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(s.tableName),
	})
	return err
}

// Close closes the store (no-op for DynamoDB)
func (s *DynamoDBStore) Close() error {
	return nil
}

// Package dynamodb implements the Store interface using AWS DynamoDB.
// This file contains shared helper functions used across operations.
package dynamodb

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// GSI key names to delete from marshaled items
var gsiKeys = []string{
	"GSI2PK", "GSI2SK",
	"GSI3PK", "GSI3SK",
	"GSI4PK", "GSI4SK",
	"GSI5PK", "GSI5SK",
}

// deleteGSIKeys removes auto-marshaled GSI keys from an item.
// This is needed because attributevalue.MarshalMap may include GSI fields
// from struct tags that we want to set manually.
func deleteGSIKeys(item map[string]types.AttributeValue) {
	for _, key := range gsiKeys {
		delete(item, key)
	}
}

// shouldSampleItem returns true for ~1% of items (deterministic sampling using cryptographic hash).
// This is used for sparse indexing in GSI5 to reduce storage costs while still
// providing debugging capability for item history.
func shouldSampleItem(itemID string) bool {
	// Use SHA256 for better distribution
	hash := sha256.Sum256([]byte(itemID))
	// Use first 8 bytes as uint64
	hashInt := binary.BigEndian.Uint64(hash[:8])
	// Check if divisible by 100 (1% probability)
	return hashInt%100 == 0
}

// setStringAttr sets a string attribute value in a DynamoDB item
func setStringAttr(item map[string]types.AttributeValue, key, value string) {
	item[key] = &types.AttributeValueMemberS{Value: value}
}

// setTTLAttr sets the TTL attribute for an item
func setTTLAttr(item map[string]types.AttributeValue, ttlUnix int64) {
	item["ttl"] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", ttlUnix)}
}

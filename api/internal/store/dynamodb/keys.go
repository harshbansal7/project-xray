// Package dynamodb implements the Store interface using AWS DynamoDB.
// This file contains key building functions and constants for consistent key construction.
package dynamodb

// Base table key prefixes
const (
	PrefixTrace = "TRACE#"
	SKTrace     = "TRACE#META"
	PrefixEvent = "EVENT#"
	PrefixDec   = "DEC#"
)

// GSI key prefixes
const (
	GSI2Prefix = "GSI2_PIPE#" // Pipeline-based queries
	GSI3Prefix = "GSI3_STEP#" // Step-type analytics
	GSI4Prefix = "GSI4_EVT#"  // Event decisions
	GSI5Prefix = "GSI5_ITEM#" // Item history
)

// TracePK returns the partition key for a trace
func TracePK(traceID string) string {
	return PrefixTrace + traceID
}

// EventSK returns the sort key for an event
func EventSK(eventID string) string {
	return PrefixEvent + eventID
}

// DecisionSK returns the sort key for a decision
func DecisionSK(eventID, decisionID string) string {
	return PrefixDec + eventID + "#" + decisionID
}

// GSI2PK returns the GSI2 partition key for pipeline queries
func GSI2PK(pipelineID string) string {
	return GSI2Prefix + pipelineID
}

// GSI3PK returns the GSI3 partition key for step-type queries
func GSI3PK(stepType string) string {
	return GSI3Prefix + stepType
}

// GSI3PKWithPipeline returns the GSI3 partition key for pipeline-specific step queries
func GSI3PKWithPipeline(stepType, pipelineID string) string {
	return GSI3Prefix + stepType + "#" + pipelineID
}

// GSI4PK returns the GSI4 partition key for event decision queries
func GSI4PK(eventID string) string {
	return GSI4Prefix + eventID
}

// GSI4SK returns the GSI4 sort key for decisions (outcome + decision_id)
func GSI4SK(outcome, decisionID string) string {
	return outcome + "#" + decisionID
}

// GSI5PK returns the GSI5 partition key for item history queries
func GSI5PK(itemID string) string {
	return GSI5Prefix + itemID
}

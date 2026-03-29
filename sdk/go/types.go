package xray

// StepType identifies a pipeline step (for example: "filter", "llm", "rank").
type StepType string

// PipelineID identifies a pipeline (for example: "competitor-selection").
type PipelineID string

// ReasonCode is a machine-queryable reason for a decision outcome.
type ReasonCode string

// CaptureMode controls how much event detail is recorded.
type CaptureMode string

const (
	// CaptureModeMetrics stores only counts, timing, and derived metrics.
	CaptureModeMetrics CaptureMode = "metrics"
	// CaptureModeSample stores a deterministic ~1% sample of decisions.
	CaptureModeSample CaptureMode = "sample"
	// CaptureModeFull stores all decisions for the event.
	CaptureModeFull CaptureMode = "full"
)

// FallbackMode controls behavior when the API is unavailable.
type FallbackMode string

const (
	// FallbackNone drops failed batches.
	FallbackNone FallbackMode = "none"
	// FallbackLocalFile writes failed batches to JSON files on disk.
	FallbackLocalFile FallbackMode = "local_file"
	// FallbackMemory keeps failed batches in memory and retries later.
	FallbackMemory FallbackMode = "memory"
)

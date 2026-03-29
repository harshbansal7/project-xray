package xray

import (
	"log"
	"os"
	"sync"
	"time"
)

// Config controls SDK behavior.
type Config struct {
	APIKey string

	Endpoint string
	Timeout  time.Duration

	AsyncSend      bool
	BatchSize      int
	FlushInterval  time.Duration
	QueueSize      int
	MaxRetry       int
	RetryBaseDelay time.Duration

	Fallback     FallbackMode
	FallbackPath string

	MaxDecisionsPerEvent int
	MaxItemSnapshotBytes int
	MaxSampleItems       int

	Enabled bool
	Debug   bool
}

func defaultConfig() Config {
	endpoint := os.Getenv("XRAY_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8080/api/v1"
	}
	return Config{
		APIKey: os.Getenv("XRAY_API_KEY"),

		Endpoint: endpoint,
		Timeout:  5 * time.Second,

		AsyncSend:      true,
		BatchSize:      100,
		FlushInterval:  time.Second,
		QueueSize:      10000,
		MaxRetry:       3,
		RetryBaseDelay: 100 * time.Millisecond,

		Fallback:     FallbackMemory,
		FallbackPath: "",

		MaxDecisionsPerEvent: 10000,
		MaxItemSnapshotBytes: 1024,
		MaxSampleItems:       5,

		Enabled: true,
		Debug:   false,
	}
}

// Option mutates config values during Configure().
type Option func(*Config)

func WithAPIKey(v string) Option               { return func(c *Config) { c.APIKey = v } }
func WithEndpoint(v string) Option             { return func(c *Config) { c.Endpoint = v } }
func WithTimeout(v time.Duration) Option       { return func(c *Config) { c.Timeout = v } }
func WithAsyncSend(v bool) Option              { return func(c *Config) { c.AsyncSend = v } }
func WithBatchSize(v int) Option               { return func(c *Config) { c.BatchSize = v } }
func WithFlushInterval(v time.Duration) Option { return func(c *Config) { c.FlushInterval = v } }
func WithQueueSize(v int) Option               { return func(c *Config) { c.QueueSize = v } }
func WithMaxRetry(v int) Option                { return func(c *Config) { c.MaxRetry = v } }
func WithRetryBaseDelay(v time.Duration) Option {
	return func(c *Config) { c.RetryBaseDelay = v }
}
func WithFallback(mode FallbackMode, path string) Option {
	return func(c *Config) {
		c.Fallback = mode
		c.FallbackPath = path
	}
}
func WithMaxDecisionsPerEvent(v int) Option { return func(c *Config) { c.MaxDecisionsPerEvent = v } }
func WithMaxItemSnapshotBytes(v int) Option { return func(c *Config) { c.MaxItemSnapshotBytes = v } }
func WithMaxSampleItems(v int) Option       { return func(c *Config) { c.MaxSampleItems = v } }
func WithEnabled(v bool) Option             { return func(c *Config) { c.Enabled = v } }
func WithDebug(v bool) Option               { return func(c *Config) { c.Debug = v } }

var (
	globalMu       sync.RWMutex
	globalConfig   = defaultConfig()
	globalClient   *Client
	pipelineReg    = map[PipelineID]map[StepType]struct{}{}
	reasonReg      = map[PipelineID]map[ReasonCode]struct{}{}
	globalReasones = map[ReasonCode]struct{}{}
)

// Configure sets SDK runtime configuration and resets the global client.
func Configure(opts ...Option) Config {
	globalMu.Lock()
	defer globalMu.Unlock()

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = time.Second
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 10000
	}
	if cfg.MaxRetry <= 0 {
		cfg.MaxRetry = 3
	}
	if cfg.RetryBaseDelay <= 0 {
		cfg.RetryBaseDelay = 100 * time.Millisecond
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.MaxDecisionsPerEvent <= 0 {
		cfg.MaxDecisionsPerEvent = 10000
	}
	if cfg.MaxItemSnapshotBytes <= 0 {
		cfg.MaxItemSnapshotBytes = 1024
	}
	if cfg.MaxSampleItems <= 0 {
		cfg.MaxSampleItems = 5
	}

	globalConfig = cfg
	if globalClient != nil {
		globalClient.Shutdown()
		globalClient = nil
	}
	return globalConfig
}

// GetConfig returns a copy of current SDK configuration.
func GetConfig() Config {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalConfig
}

// Shutdown flushes and closes the global background client.
func Shutdown() {
	globalMu.Lock()
	defer globalMu.Unlock()
	if globalClient != nil {
		globalClient.Shutdown()
		globalClient = nil
	}
}

func getClient() *Client {
	globalMu.Lock()
	defer globalMu.Unlock()
	if globalClient == nil {
		globalClient = NewClient(globalConfig)
	}
	return globalClient
}

// RegisterPipeline registers allowed step types and optional reason codes for a pipeline.
func RegisterPipeline(pipelineID PipelineID, stepTypes []StepType, reasonCodes []ReasonCode) {
	globalMu.Lock()
	defer globalMu.Unlock()

	stepSet := make(map[StepType]struct{}, len(stepTypes))
	for _, s := range stepTypes {
		stepSet[s] = struct{}{}
	}
	pipelineReg[pipelineID] = stepSet

	if len(reasonCodes) > 0 {
		rset := make(map[ReasonCode]struct{}, len(reasonCodes))
		for _, r := range reasonCodes {
			rset[r] = struct{}{}
		}
		reasonReg[pipelineID] = rset
	}

	if globalConfig.Debug {
		log.Printf("[xray] registered pipeline=%s step_types=%d reason_codes=%d", pipelineID, len(stepTypes), len(reasonCodes))
	}
}

// RegisterReasonCodes registers global reason codes shared by multiple pipelines.
func RegisterReasonCodes(reasonCodes []ReasonCode) {
	globalMu.Lock()
	defer globalMu.Unlock()
	for _, r := range reasonCodes {
		globalReasones[r] = struct{}{}
	}
}

// IsPipelineRegistered checks whether a pipeline is registered.
func IsPipelineRegistered(pipelineID PipelineID) bool {
	globalMu.RLock()
	defer globalMu.RUnlock()
	_, ok := pipelineReg[pipelineID]
	return ok
}

func validateStepType(pipelineID PipelineID, stepType StepType) error {
	globalMu.RLock()
	defer globalMu.RUnlock()
	steps, ok := pipelineReg[pipelineID]
	if !ok {
		return nil
	}
	if _, ok := steps[stepType]; !ok {
		return &ValidationError{Message: "step_type is not registered for this pipeline", Field: "step_type"}
	}
	return nil
}

func validateReasonCode(pipelineID PipelineID, reason *string, strict bool) error {
	if reason == nil || *reason == "" {
		return nil
	}
	r := ReasonCode(*reason)
	globalMu.RLock()
	defer globalMu.RUnlock()
	if _, ok := globalReasones[r]; ok {
		return nil
	}
	if rs, ok := reasonReg[pipelineID]; ok {
		if _, ok := rs[r]; ok {
			return nil
		}
	}
	if strict {
		return &ValidationError{Message: "reason_code is not registered", Field: "reason_code"}
	}
	if globalConfig.Debug {
		log.Printf("[xray] warning: unregistered reason_code=%s pipeline=%s", *reason, pipelineID)
	}
	return nil
}

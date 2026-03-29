package xray

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"time"
)

// EventOptions configures event instrumentation behavior.
type EventOptions struct {
	CaptureMode     CaptureMode
	ParentEventID   *string
	Annotations     map[string]interface{}
	SamplingConfig  *SamplingConfig
	StrictReasoning bool
}

// DecisionOptions carries optional decision metadata.
type DecisionOptions struct {
	ReasonCode   *string
	ReasonDetail *string
	Scores       map[string]float64
	ItemSnapshot map[string]interface{}
	Timestamp    *time.Time
}

// Event represents one trace step with optional item-level decisions.
type Event struct {
	trace         *Trace
	data          EventData
	decisions     []DecisionData
	cfg           Config
	strictReasons bool
	closed        bool
}

func newEvent(t *Trace, stepType StepType, opts EventOptions) *Event {
	mode := opts.CaptureMode
	if mode == "" {
		mode = CaptureModeMetrics
	}
	s := opts.SamplingConfig
	if s == nil {
		s = t.samplingCfg
	}
	return &Event{
		trace: t,
		data: EventData{
			EventID:       newID(),
			TraceID:       t.data.TraceID,
			ParentEventID: opts.ParentEventID,
			StepType:      string(stepType),
			CaptureMode:   mode,
			Annotations:   cloneMap(opts.Annotations),
			StartedAt:     time.Now().UTC(),
		},
		decisions:     []DecisionData{},
		cfg:           t.cfg,
		strictReasons: opts.StrictReasoning,
	}
}

func (e *Event) validate() error {
	if e.data.StepType == "" {
		return &ValidationError{Field: "step_type", Message: "cannot be empty"}
	}
	if e.data.CaptureMode != CaptureModeMetrics && e.data.CaptureMode != CaptureModeSample && e.data.CaptureMode != CaptureModeFull {
		return &ValidationError{Field: "capture_mode", Message: "must be one of metrics|sample|full"}
	}
	return validateStepType(e.trace.pipelineID, StepType(e.data.StepType))
}

// ID returns event_id. It becomes non-empty after first serialization/send.
func (e *Event) ID() string {
	return e.data.EventID
}

// SetInput records event input count and optional sample.
func (e *Event) SetInput(data interface{}, explicitCount ...int) {
	count := deriveCount(data, explicitCount...)
	e.data.InputCount = &count
	e.data.InputSample = sampleItems(data, e.cfg.MaxSampleItems)
}

// SetOutput records event output count and optional sample.
func (e *Event) SetOutput(data interface{}, explicitCount ...int) {
	count := deriveCount(data, explicitCount...)
	e.data.OutputCount = &count
	e.data.OutputSample = sampleItems(data, e.cfg.MaxSampleItems)
}

// Annotate attaches key/value metadata to the event.
func (e *Event) Annotate(key string, value interface{}) {
	if e.data.Annotations == nil {
		e.data.Annotations = map[string]interface{}{}
	}
	e.data.Annotations[key] = value
}

// RecordDecision records one item-level decision for this event.
func (e *Event) RecordDecision(itemID, outcome string, opts DecisionOptions) error {
	if itemID == "" {
		return &ValidationError{Field: "item_id", Message: "cannot be empty"}
	}
	if outcome == "" {
		return &ValidationError{Field: "outcome", Message: "cannot be empty"}
	}

	if !e.shouldRecordDecision(outcome, itemID, e.trace.samplingCfg) {
		return nil
	}
	if len(e.decisions) >= e.cfg.MaxDecisionsPerEvent {
		return nil
	}

	if err := validateReasonCode(e.trace.pipelineID, opts.ReasonCode, e.strictReasons); err != nil {
		return err
	}

	timestamp := time.Now().UTC()
	if opts.Timestamp != nil {
		timestamp = opts.Timestamp.UTC()
	}

	d := DecisionData{
		DecisionID:   newID(),
		EventID:      e.data.EventID,
		TraceID:      e.data.TraceID,
		ItemID:       itemID,
		Outcome:      outcome,
		ReasonCode:   opts.ReasonCode,
		ReasonDetail: opts.ReasonDetail,
		Scores:       copyScores(opts.Scores),
		ItemSnapshot: e.truncateSnapshot(opts.ItemSnapshot),
		Timestamp:    timestamp,
	}
	e.decisions = append(e.decisions, d)
	return nil
}

// End finalizes event metrics and queues event + decisions.
func (e *Event) End(runErr error) {
	if e.closed {
		return
	}
	now := time.Now().UTC()
	e.data.EndedAt = &now
	e.computeMetrics()

	if e.cfg.Enabled {
		e.trace.client.QueueEvent(e.data)
		if len(e.decisions) > 0 {
			for i := range e.decisions {
				e.decisions[i].EventID = e.data.EventID
				e.decisions[i].TraceID = e.data.TraceID
			}
			e.trace.client.QueueDecisions(e.decisions)
		}
	}
	e.closed = true
}

func (e *Event) computeMetrics() {
	if e.data.StartedAt.IsZero() || e.data.EndedAt == nil {
		return
	}
	dur := e.data.EndedAt.Sub(e.data.StartedAt).Seconds() * 1000
	e.data.Metrics.DurationMS = &dur
	e.data.Metrics.InputCount = e.data.InputCount
	e.data.Metrics.OutputCount = e.data.OutputCount
	if e.data.InputCount != nil && e.data.OutputCount != nil && *e.data.InputCount > 0 {
		r := 1.0 - float64(*e.data.OutputCount)/float64(*e.data.InputCount)
		e.data.Metrics.ReductionRatio = &r
	}
}

func (e *Event) truncateSnapshot(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}
	b, err := json.Marshal(in)
	if err != nil {
		return map[string]interface{}{"_error": "snapshot_not_serializable"}
	}
	if len(b) <= e.cfg.MaxItemSnapshotBytes {
		return in
	}
	return map[string]interface{}{
		"_truncated":     true,
		"_original_size": len(b),
		"preview":        string(b[:e.cfg.MaxItemSnapshotBytes]),
	}
}

func (e *Event) shouldRecordDecision(outcome, itemID string, traceSampling *SamplingConfig) bool {
	if traceSampling != nil {
		return traceSampling.ShouldSample(outcome, itemID)
	}
	switch e.data.CaptureMode {
	case CaptureModeMetrics:
		return false
	case CaptureModeFull:
		return true
	case CaptureModeSample:
		h := md5.Sum([]byte(itemID))
		v := binary.BigEndian.Uint32(h[:4])
		return v%100 == 0
	default:
		return false
	}
}

func deriveCount(data interface{}, explicitCount ...int) int {
	if len(explicitCount) > 0 {
		return explicitCount[0]
	}
	if data == nil {
		return 0
	}
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return v.Len()
	default:
		return 1
	}
}

func sampleItems(data interface{}, max int) []interface{} {
	if data == nil || max <= 0 {
		return nil
	}
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	// If not an array/slice, wrap single object in an array for visualization
	if v.Kind() != reflect.Array && v.Kind() != reflect.Slice {
		// For maps, structs, and other single objects, wrap them
		return []interface{}{data}
	}
	
	n := v.Len()
	if n == 0 {
		return nil
	}
	if n <= max {
		return toInterfaceSlice(v)
	}
	step := n / max
	if step == 0 {
		step = 1
	}
	out := make([]interface{}, 0, max)
	for i := 0; i < n && len(out) < max; i += step {
		out = append(out, v.Index(i).Interface())
	}
	return out
}

func toInterfaceSlice(v reflect.Value) []interface{} {
	out := make([]interface{}, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		out = append(out, v.Index(i).Interface())
	}
	return out
}

func copyScores(in map[string]float64) map[string]float64 {
	if in == nil {
		return nil
	}
	out := make(map[string]float64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

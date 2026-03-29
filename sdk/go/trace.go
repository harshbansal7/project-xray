package xray

import (
	"context"
	"time"
)

// TraceOptions configures a trace.
type TraceOptions struct {
	InputData       interface{}
	Metadata        map[string]interface{}
	Tags            []string
	SamplingConfig  *SamplingConfig
	TraceIDOverride string
}

// Trace is one complete pipeline execution.
type Trace struct {
	data        TraceData
	pipelineID  PipelineID
	client      *Client
	cfg         Config
	samplingCfg *SamplingConfig
	closed      bool
}

// StartTrace starts a trace and immediately queues its start record.
func StartTrace(pipelineID PipelineID, opts TraceOptions) (*Trace, error) {
	cfg := GetConfig()
	traceID := opts.TraceIDOverride
	if traceID == "" {
		traceID = newID()
	}
	t := &Trace{
		data: TraceData{
			TraceID:    traceID,
			PipelineID: string(pipelineID),
			StartedAt:  time.Now().UTC(),
			Status:     "running",
			Metadata:   cloneMap(opts.Metadata),
			InputData:  opts.InputData,
			Tags:       append([]string{}, opts.Tags...),
		},
		pipelineID:  pipelineID,
		client:      getClient(),
		cfg:         cfg,
		samplingCfg: opts.SamplingConfig,
	}
	if cfg.Enabled {
		t.client.QueueTraceStart(t.data)
	}
	return t, nil
}

// ID returns trace_id for this trace.
func (t *Trace) ID() string {
	return t.data.TraceID
}

// End marks trace as completed/failed and queues its final state.
func (t *Trace) End(runErr error) {
	if t.closed {
		return
	}
	now := time.Now().UTC()
	t.data.EndedAt = &now
	if runErr != nil {
		t.data.Status = "failed"
	} else {
		t.data.Status = "completed"
	}
	if t.cfg.Enabled {
		t.client.QueueTraceEnd(t.data)
	}
	t.closed = true
}

// StartEvent starts a pipeline event under this trace.
func (t *Trace) StartEvent(stepType StepType, opts EventOptions) (*Event, error) {
	event := newEvent(t, stepType, opts)
	if err := event.validate(); err != nil {
		return nil, err
	}
	return event, nil
}

// WithEvent is a helper to ensure End() is always called.
func (t *Trace) WithEvent(stepType StepType, opts EventOptions, fn func(*Event) error) error {
	e, err := t.StartEvent(stepType, opts)
	if err != nil {
		return err
	}
	err = fn(e)
	e.End(err)
	return err
}

// WithTrace is a helper to ensure End() is always called.
func WithTrace(ctx context.Context, pipelineID PipelineID, opts TraceOptions, fn func(context.Context, *Trace) error) error {
	tr, err := StartTrace(pipelineID, opts)
	if err != nil {
		return err
	}
	err = fn(ctx, tr)
	tr.End(err)
	return err
}

func cloneMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

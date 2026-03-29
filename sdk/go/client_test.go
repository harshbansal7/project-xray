package xray

import "testing"

func TestSamplingConfigShouldSample(t *testing.T) {
	s := &SamplingConfig{OutcomeRates: map[string]float64{"accepted": 0.0, "rejected": 1.0, "*": 0.5}}
	if s.ShouldSample("accepted", "item-1") {
		t.Fatalf("accepted should never sample at 0.0")
	}
	if !s.ShouldSample("rejected", "item-1") {
		t.Fatalf("rejected should always sample at 1.0")
	}
}

func TestRegisterPipelineValidation(t *testing.T) {
	Configure()
	RegisterPipeline("p1", []StepType{"filter"}, nil)
	if err := validateStepType("p1", "rank"); err == nil {
		t.Fatalf("expected validation error for unregistered step")
	}
	if err := validateStepType("p1", "filter"); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

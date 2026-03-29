package main

import (
	"context"
	"fmt"
	"log"
	"time"

	xray "github.com/xray-sdk/xray-go"
)

const (
	pipelineID xray.PipelineID = "competitor-selection"
	stepFilter xray.StepType   = "filter"
	stepRank   xray.StepType   = "rank"
)

func main() {
	xray.Configure(
		xray.WithEndpoint("http://localhost:8080/api/v1"),
		xray.WithAsyncSend(true),
		xray.WithDebug(true),
		xray.WithFallback(xray.FallbackLocalFile, "./xray_go_fallback"),
	)
	defer xray.Shutdown()

	xray.RegisterPipeline(
		pipelineID,
		[]xray.StepType{stepFilter, stepRank},
		[]xray.ReasonCode{"PRICE_TOO_HIGH", "CATEGORY_MISMATCH", "HIGH_RELEVANCE", "PASSED_ALL_FILTERS"},
	)

	sampling := &xray.SamplingConfig{OutcomeRates: map[string]float64{
		"rejected": 1.0,
		"accepted": 0.2,
		"*":        0.05,
	}}

	trace, err := xray.StartTrace(pipelineID, xray.TraceOptions{
		Metadata:       map[string]interface{}{"source_asin": "B09V3KXJPB", "env": "dev"},
		Tags:           []string{"demo", "go-sdk"},
		SamplingConfig: sampling,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer trace.End(nil)

	candidates := []map[string]interface{}{
		{"asin": "B012345678", "price": 1400.0, "category": "electronics"},
		{"asin": "B045678901", "price": 2200.0, "category": "electronics"},
		{"asin": "B067890123", "price": 999.0, "category": "office"},
	}

	filterEvent, err := trace.StartEvent(stepFilter, xray.EventOptions{CaptureMode: xray.CaptureModeFull})
	if err != nil {
		log.Fatal(err)
	}
	filterEvent.SetInput(candidates)

	passed := make([]map[string]interface{}, 0, len(candidates))
	for _, c := range candidates {
		asin := c["asin"].(string)
		price := c["price"].(float64)
		category := c["category"].(string)

		if price > 1500 {
			reason := "PRICE_TOO_HIGH"
			detail := "price exceeds threshold 1500"
			_ = filterEvent.RecordDecision(asin, "rejected", xray.DecisionOptions{ReasonCode: &reason, ReasonDetail: &detail, Scores: map[string]float64{"price": price}})
			continue
		}
		if category != "electronics" {
			reason := "CATEGORY_MISMATCH"
			detail := "category must be electronics"
			_ = filterEvent.RecordDecision(asin, "rejected", xray.DecisionOptions{ReasonCode: &reason, ReasonDetail: &detail})
			continue
		}

		reason := "PASSED_ALL_FILTERS"
		detail := "passed price and category checks"
		_ = filterEvent.RecordDecision(asin, "accepted", xray.DecisionOptions{ReasonCode: &reason, ReasonDetail: &detail})
		passed = append(passed, c)
	}
	filterEvent.SetOutput(passed)
	filterEvent.Annotate("rejected_count", len(candidates)-len(passed))
	filterEvent.End(nil)

	rankEvent, err := trace.StartEvent(stepRank, xray.EventOptions{CaptureMode: xray.CaptureModeSample})
	if err != nil {
		log.Fatal(err)
	}
	rankEvent.SetInput(passed)
	for i, c := range passed {
		score := 0.9 - float64(i)*0.1
		reason := "HIGH_RELEVANCE"
		detail := "ranked by relevance score"
		_ = rankEvent.RecordDecision(c["asin"].(string), "accepted", xray.DecisionOptions{ReasonCode: &reason, ReasonDetail: &detail, Scores: map[string]float64{"relevance_score": score}})
	}
	rankEvent.SetOutput(passed)
	rankEvent.End(nil)

	time.Sleep(2 * time.Second)

	if trace.ID() != "" {
		data, err := xray.GetTrace(context.Background(), trace.ID())
		if err == nil {
			fmt.Printf("Fetched trace %v with %d events\n", data.Trace["trace_id"], len(data.Events))
		}
	}

	fmt.Printf("Trace completed. trace_id=%s\n", trace.ID())
}

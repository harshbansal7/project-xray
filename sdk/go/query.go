package xray

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// QueryOptions provides flexible filtering for SDK-side query helpers.
type QueryOptions struct {
	PipelineID        string
	StepType          string
	MinReductionRatio *float64
	Tags              []string
	Limit             int
	Cursor            string
}

// DecisionQueryOptions filters event decisions.
type DecisionQueryOptions struct {
	Outcome    string
	ReasonCode string
	ItemID     string
	Limit      int
	Cursor     string
}

// GetTrace fetches one trace with events/decisions from the API.
func GetTrace(ctx context.Context, traceID string) (*TraceWithEvents, error) {
	var out TraceWithEvents
	if err := doJSON(ctx, http.MethodGet, "/traces/"+url.PathEscape(traceID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Query runs GET /query/events when step filters are present, otherwise GET /traces.
func Query(ctx context.Context, opts QueryOptions) (map[string]interface{}, error) {
	params := url.Values{}
	if opts.PipelineID != "" {
		params.Set("pipeline_id", opts.PipelineID)
	}
	if opts.StepType != "" {
		params.Set("step_type", opts.StepType)
	}
	if opts.MinReductionRatio != nil {
		params.Set("min_reduction_ratio", strconv.FormatFloat(*opts.MinReductionRatio, 'f', -1, 64))
	}
	if len(opts.Tags) > 0 {
		params.Set("tags", strings.Join(opts.Tags, ","))
	}
	if opts.Limit > 0 {
		params.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Cursor != "" {
		params.Set("cursor", opts.Cursor)
	}

	path := "/traces"
	if opts.StepType != "" || opts.MinReductionRatio != nil {
		path = "/query/events"
	}
	if qs := params.Encode(); qs != "" {
		path += "?" + qs
	}

	var out map[string]interface{}
	if err := doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// QueryAdvanced posts directly to /query for advanced server-side querying.
func QueryAdvanced(ctx context.Context, req QueryRequest) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := doJSON(ctx, http.MethodPost, "/query", req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetDecisions fetches event decisions with optional filters.
func GetDecisions(ctx context.Context, traceID, eventID string, opts DecisionQueryOptions) (map[string]interface{}, error) {
	params := url.Values{}
	if opts.Outcome != "" {
		params.Set("outcome", opts.Outcome)
	}
	if opts.ReasonCode != "" {
		params.Set("reason_code", opts.ReasonCode)
	}
	if opts.ItemID != "" {
		params.Set("item_id", opts.ItemID)
	}
	if opts.Limit > 0 {
		params.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Cursor != "" {
		params.Set("cursor", opts.Cursor)
	}
	path := fmt.Sprintf("/traces/%s/events/%s/decisions", url.PathEscape(traceID), url.PathEscape(eventID))
	if qs := params.Encode(); qs != "" {
		path += "?" + qs
	}
	var out map[string]interface{}
	if err := doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetItemHistory fetches all known decisions for one item across traces.
func GetItemHistory(ctx context.Context, itemID string, limit int) (map[string]interface{}, error) {
	path := "/items/" + url.PathEscape(itemID) + "/history"
	if limit > 0 {
		path += "?limit=" + strconv.Itoa(limit)
	}
	var out map[string]interface{}
	if err := doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func doJSON(ctx context.Context, method, path string, reqBody interface{}, out interface{}) error {
	cfg := GetConfig()
	base := strings.TrimRight(cfg.Endpoint, "/")
	fullURL := base + path

	var body io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	client := &http.Client{Timeout: cfg.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}
	if out == nil {
		return nil
	}
	if len(respBody) == 0 {
		return nil
	}
	return json.Unmarshal(respBody, out)
}

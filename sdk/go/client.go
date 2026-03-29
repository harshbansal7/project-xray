package xray

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type itemType string

const (
	itemTrace    itemType = "trace"
	itemEvent    itemType = "event"
	itemDecision itemType = "decision"
)

type queueItem struct {
	Type itemType    `json:"type"`
	Data interface{} `json:"data"`
}

// Client is the transport layer for async/sync ingestion.
type Client struct {
	cfg      Config
	http     *http.Client
	baseURL  string
	headers  map[string]string
	queue    chan queueItem
	shutdown chan struct{}
	done     chan struct{}

	fallbackMu     sync.Mutex
	memoryFallback []queueItem
}

// NewClient creates a new X-Ray ingestion client.
func NewClient(cfg Config) *Client {
	base := strings.TrimRight(cfg.Endpoint, "/")
	c := &Client{
		cfg:      cfg,
		http:     &http.Client{Timeout: cfg.Timeout},
		baseURL:  base,
		headers:  map[string]string{"Content-Type": "application/json"},
		queue:    make(chan queueItem, cfg.QueueSize),
		shutdown: make(chan struct{}),
		done:     make(chan struct{}),
	}
	if cfg.APIKey != "" {
		c.headers["Authorization"] = "Bearer " + cfg.APIKey
	}
	if cfg.AsyncSend {
		go c.loop()
	}
	return c
}

// QueueTraceStart queues trace start data.
func (c *Client) QueueTraceStart(trace TraceData) {
	c.enqueue(queueItem{Type: itemTrace, Data: trace})
}

// QueueTraceEnd queues trace end data.
func (c *Client) QueueTraceEnd(trace TraceData) {
	c.enqueue(queueItem{Type: itemTrace, Data: trace})
}

// QueueEvent queues event data.
func (c *Client) QueueEvent(event EventData) {
	c.enqueue(queueItem{Type: itemEvent, Data: event})
}

// QueueDecisions queues decisions data.
func (c *Client) QueueDecisions(decisions []DecisionData) {
	for _, d := range decisions {
		c.enqueue(queueItem{Type: itemDecision, Data: d})
	}
}

func (c *Client) enqueue(item queueItem) {
	if !c.cfg.Enabled {
		return
	}
	if !c.cfg.AsyncSend {
		if err := c.sendBatch([]queueItem{item}); err != nil {
			c.handleFailure([]queueItem{item}, err)
		}
		return
	}

	select {
	case c.queue <- item:
	default:
		c.handleFailure([]queueItem{item}, fmt.Errorf("queue is full"))
	}
}

func (c *Client) loop() {
	defer close(c.done)
	ticker := time.NewTicker(c.cfg.FlushInterval)
	defer ticker.Stop()

	batch := make([]queueItem, 0, c.cfg.BatchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := c.sendBatch(batch); err != nil {
			c.handleFailure(batch, err)
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-c.shutdown:
			drain := true
			for drain {
				select {
				case it := <-c.queue:
					batch = append(batch, it)
				default:
					drain = false
				}
			}
			flush()
			return
		case it := <-c.queue:
			batch = append(batch, it)
			if len(batch) >= c.cfg.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (c *Client) sendBatch(batch []queueItem) error {
	if len(batch) == 0 {
		return nil
	}

	pending := c.takeMemoryFallback()
	if len(pending) > 0 {
		batch = append(pending, batch...)
	}

	traceMap := map[string]TraceData{}
	eventMap := map[string]EventData{}
	decisions := make([]DecisionData, 0)

	for _, it := range batch {
		switch it.Type {
		case itemTrace:
			b, _ := json.Marshal(it.Data)
			var t TraceData
			if err := json.Unmarshal(b, &t); err == nil {
				if t.TraceID != "" {
					traceMap[t.TraceID] = t
				}
			}
		case itemEvent:
			b, _ := json.Marshal(it.Data)
			var e EventData
			if err := json.Unmarshal(b, &e); err == nil {
				if e.EventID != "" {
					eventMap[e.EventID] = e
				}
			}
		case itemDecision:
			b, _ := json.Marshal(it.Data)
			var d DecisionData
			if err := json.Unmarshal(b, &d); err == nil {
				decisions = append(decisions, d)
			}
		}
	}

	traces := make([]TraceData, 0, len(traceMap))
	for _, t := range traceMap {
		traces = append(traces, t)
	}
	events := make([]EventData, 0, len(eventMap))
	for _, e := range eventMap {
		events = append(events, e)
	}

	if len(traces) > 0 {
		if err := c.postJSON("/traces/batch", map[string]interface{}{"traces": traces}); err != nil {
			return err
		}
	}
	if len(events) > 0 {
		if err := c.postJSON("/events/batch", map[string]interface{}{"events": events}); err != nil {
			return err
		}
	}
	if len(decisions) > 0 {
		if err := c.postJSON("/decisions/batch", map[string]interface{}{"decisions": decisions}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) postJSON(path string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := c.baseURL + path
	var last error
	for attempt := 0; attempt < c.cfg.MaxRetry; attempt++ {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		for k, v := range c.headers {
			req.Header.Set(k, v)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			last = err
		} else {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
			if resp.StatusCode < 500 {
				return &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
			}
			last = &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
		}
		time.Sleep(c.cfg.RetryBaseDelay * time.Duration(1<<attempt))
	}
	return last
}

func (c *Client) handleFailure(batch []queueItem, err error) {
	if c.cfg.Debug {
		log.Printf("[xray] failed to send batch (%d items): %v", len(batch), err)
	}
	switch c.cfg.Fallback {
	case FallbackNone:
		return
	case FallbackLocalFile:
		c.writeFallback(batch)
	case FallbackMemory:
		c.pushMemoryFallback(batch)
	}
}

func (c *Client) pushMemoryFallback(batch []queueItem) {
	c.fallbackMu.Lock()
	defer c.fallbackMu.Unlock()
	max := c.cfg.QueueSize
	if max < 1000 {
		max = 1000
	}
	if len(c.memoryFallback)+len(batch) > max {
		over := len(c.memoryFallback) + len(batch) - max
		if over < len(c.memoryFallback) {
			c.memoryFallback = c.memoryFallback[over:]
		} else {
			c.memoryFallback = nil
		}
	}
	c.memoryFallback = append(c.memoryFallback, batch...)
}

func (c *Client) takeMemoryFallback() []queueItem {
	c.fallbackMu.Lock()
	defer c.fallbackMu.Unlock()
	if len(c.memoryFallback) == 0 {
		return nil
	}
	out := make([]queueItem, len(c.memoryFallback))
	copy(out, c.memoryFallback)
	c.memoryFallback = nil
	return out
}

func (c *Client) writeFallback(batch []queueItem) {
	if c.cfg.FallbackPath == "" {
		return
	}
	if err := os.MkdirAll(c.cfg.FallbackPath, 0o755); err != nil {
		return
	}
	name := fmt.Sprintf("xray_%d.json", time.Now().UnixNano()/1e6)
	path := filepath.Join(c.cfg.FallbackPath, name)
	data, err := json.MarshalIndent(batch, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

// Shutdown flushes pending buffered data and stops the background worker.
func (c *Client) Shutdown() {
	if !c.cfg.AsyncSend {
		return
	}
	close(c.shutdown)
	<-c.done
}

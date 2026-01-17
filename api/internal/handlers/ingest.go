package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/xray-sdk/xray-api/internal/models"
	"github.com/xray-sdk/xray-api/internal/store"
)

// parseTime tries multiple time formats to handle Python datetime serialization
// which includes microseconds (RFC3339Nano) that standard RFC3339 doesn't handle
func parseTime(s string) (time.Time, error) {
	// Try RFC3339Nano first (handles microseconds like "2026-01-05T22:21:11.908846+05:30")
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	// Fall back to standard RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try ISO format without timezone (Python datetime default: "2026-01-05T22:21:11.908846")
	if t, err := time.Parse("2006-01-02T15:04:05.999999", s); err == nil {
		return t.UTC(), nil
	}
	// Try ISO format without microseconds and without timezone
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("could not parse time: %s", s)
}

// IngestHandler handles data ingestion endpoints
type IngestHandler struct {
	store store.Store
}

// NewIngestHandler creates a new ingest handler
func NewIngestHandler(s store.Store) *IngestHandler {
	return &IngestHandler{store: s}
}

// CreateTrace handles POST /api/v1/traces
// @Summary Create a new trace
// @Tags traces
// @Accept json
// @Produce json
// @Param trace body models.CreateTraceRequest true "Trace data"
// @Success 201 {object} models.APIResponse
// @Failure 400 {object} models.ErrorResponse
// @Router /traces [post]
func (h *IngestHandler) CreateTrace(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTraceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Parse time
	startedAt, err := parseTime(req.StartedAt)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid started_at format", "use RFC3339")
		return
	}

	// Generate ID if not provided
	traceID := req.TraceID
	if traceID == "" {
		traceID = uuid.New().String()
	}

	trace := &models.Trace{
		TraceID:    traceID,
		PipelineID: req.PipelineID,
		StartedAt:  startedAt,
		Status:     "running",
		Metadata:   req.Metadata,
		InputData:  req.InputData,
		Tags:       req.Tags,
	}

	if err := h.store.CreateTrace(r.Context(), trace); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create trace", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, models.APIResponse{
		Status: "created",
		Data:   map[string]string{"trace_id": traceID},
	})
}

// BatchCreateTraces handles POST /api/v1/traces/batch
// @Summary Create multiple traces
// @Tags traces
// @Accept json
// @Produce json
// @Param traces body models.BatchTracesRequest true "Batch traces"
// @Success 201 {object} models.APIResponse
// @Router /traces/batch [post]
func (h *IngestHandler) BatchCreateTraces(w http.ResponseWriter, r *http.Request) {
	var req models.BatchTracesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	traces := make([]*models.Trace, len(req.Traces))
	for i, t := range req.Traces {
		startedAt := time.Now()
		if parsedTime, err := parseTime(t.StartedAt); err == nil {
			startedAt = parsedTime
		}
		traceID := t.TraceID
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// Use status from request, default to "running"
		status := "running"
		if t.Status != "" {
			status = t.Status
		}

		// Parse ended_at if provided
		var endedAt *time.Time
		if t.EndedAt != nil {
			if parsedEnd, err := parseTime(*t.EndedAt); err == nil {
				endedAt = &parsedEnd
			}
		}

		traces[i] = &models.Trace{
			TraceID:    traceID,
			PipelineID: t.PipelineID,
			StartedAt:  startedAt,
			EndedAt:    endedAt,
			Status:     status,
			Metadata:   t.Metadata,
			InputData:  t.InputData,
			Tags:       t.Tags,
		}
	}

	if err := h.store.BatchCreateTraces(r.Context(), traces); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to batch create traces", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, models.APIResponse{
		Status: "created",
		Data:   map[string]int{"count": len(traces)},
	})
}

// UpdateTrace handles PATCH /api/v1/traces/{traceId}
// @Summary Update a trace
// @Tags traces
// @Accept json
// @Produce json
// @Param traceId path string true "Trace ID"
// @Param updates body models.UpdateTraceRequest true "Updates"
// @Success 200 {object} models.APIResponse
// @Router /traces/{traceId} [patch]
func (h *IngestHandler) UpdateTrace(w http.ResponseWriter, r *http.Request) {
	traceID := chi.URLParam(r, "traceId")

	var req models.UpdateTraceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	updates := &store.TraceUpdates{}
	if req.EndedAt != nil {
		if t, err := parseTime(*req.EndedAt); err == nil {
			updates.EndedAt = &t
		}
	}
	updates.Status = req.Status

	if err := h.store.UpdateTrace(r.Context(), traceID, updates); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update trace", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, models.APIResponse{Status: "updated"})
}

// CreateEvent handles POST /api/v1/events
// @Summary Create a new event
// @Tags events
// @Accept json
// @Produce json
// @Param event body models.CreateEventRequest true "Event data"
// @Success 201 {object} models.APIResponse
// @Router /events [post]
func (h *IngestHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req models.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	startedAt := time.Now()
	if parsedTime, err := parseTime(req.StartedAt); err == nil {
		startedAt = parsedTime
	}

	eventID := req.EventID
	if eventID == "" {
		eventID = uuid.New().String()
	}

	captureMode := models.CaptureMode(req.CaptureMode)
	if captureMode == "" {
		captureMode = models.CaptureModeMetrics
	}

	event := &models.Event{
		EventID:       eventID,
		TraceID:       req.TraceID,
		ParentEventID: req.ParentEventID,
		StepName:      req.StepName,
		StepType:      models.StepType(req.StepType),
		CaptureMode:   captureMode,
		InputCount:    req.InputCount,
		InputSample:   req.InputSample,
		OutputCount:   req.OutputCount,
		OutputSample:  req.OutputSample,
		Annotations:   req.Annotations,
		StartedAt:     startedAt,
	}

	if req.EndedAt != nil {
		if t, err := parseTime(*req.EndedAt); err == nil {
			event.EndedAt = &t
		}
	}

	if req.Metrics != nil {
		event.Metrics = *req.Metrics
	}

	if err := h.store.CreateEvent(r.Context(), event); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create event", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, models.APIResponse{
		Status: "created",
		Data:   map[string]string{"event_id": eventID},
	})
}

// BatchCreateEvents handles POST /api/v1/events/batch
// @Summary Create multiple events
// @Tags events
// @Accept json
// @Produce json
// @Param events body models.BatchEventsRequest true "Batch events"
// @Success 201 {object} models.APIResponse
// @Router /events/batch [post]
func (h *IngestHandler) BatchCreateEvents(w http.ResponseWriter, r *http.Request) {
	var req models.BatchEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	events := make([]*models.Event, len(req.Events))
	for i, e := range req.Events {
		startedAt := time.Now()
		if parsedTime, err := parseTime(e.StartedAt); err == nil {
			startedAt = parsedTime
		}
		eventID := e.EventID
		if eventID == "" {
			eventID = uuid.New().String()
		}

		captureMode := models.CaptureMode(e.CaptureMode)
		if captureMode == "" {
			captureMode = models.CaptureModeMetrics
		}

		events[i] = &models.Event{
			EventID:       eventID,
			TraceID:       e.TraceID,
			ParentEventID: e.ParentEventID,
			StepName:      e.StepName,
			StepType:      models.StepType(e.StepType),
			CaptureMode:   captureMode,
			InputCount:    e.InputCount,
			OutputCount:   e.OutputCount,
			Annotations:   e.Annotations,
			StartedAt:     startedAt,
		}

		if e.EndedAt != nil {
			if t, err := parseTime(*e.EndedAt); err == nil {
				events[i].EndedAt = &t
			}
		}

		if e.Metrics != nil {
			events[i].Metrics = *e.Metrics
		}
	}

	if err := h.store.BatchCreateEvents(r.Context(), events); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to batch create events", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, models.APIResponse{
		Status: "created",
		Data:   map[string]int{"count": len(events)},
	})
}

// CreateDecision handles POST /api/v1/decisions
// @Summary Create a new decision
// @Tags decisions
// @Accept json
// @Produce json
// @Param decision body models.CreateDecisionRequest true "Decision data"
// @Success 201 {object} models.APIResponse
// @Router /decisions [post]
func (h *IngestHandler) CreateDecision(w http.ResponseWriter, r *http.Request) {
	var req models.CreateDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	decisionID := req.DecisionID
	if decisionID == "" {
		decisionID = uuid.New().String()
	}

	timestamp := time.Now()
	if req.Timestamp != nil {
		if t, err := parseTime(*req.Timestamp); err == nil {
			timestamp = t
		}
	}

	decision := &models.Decision{
		DecisionID:   decisionID,
		EventID:      req.EventID,
		TraceID:      req.TraceID,
		ItemID:       req.ItemID,
		Outcome:      req.Outcome,
		ReasonCode:   req.ReasonCode,
		ReasonDetail: req.ReasonDetail,
		Scores:       req.Scores,
		ItemSnapshot: req.ItemSnapshot,
		Timestamp:    timestamp,
	}

	if err := h.store.CreateDecision(r.Context(), decision); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create decision", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, models.APIResponse{
		Status: "created",
		Data:   map[string]string{"decision_id": decisionID},
	})
}

// BatchCreateDecisions handles POST /api/v1/decisions/batch
// @Summary Create multiple decisions
// @Tags decisions
// @Accept json
// @Produce json
// @Param decisions body models.BatchDecisionsRequest true "Batch decisions"
// @Success 201 {object} models.APIResponse
// @Router /decisions/batch [post]
func (h *IngestHandler) BatchCreateDecisions(w http.ResponseWriter, r *http.Request) {
	var req models.BatchDecisionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	decisions := make([]*models.Decision, len(req.Decisions))
	for i, d := range req.Decisions {
		decisionID := d.DecisionID
		if decisionID == "" {
			decisionID = uuid.New().String()
		}

		timestamp := time.Now()
		if d.Timestamp != nil {
			if t, err := parseTime(*d.Timestamp); err == nil {
				timestamp = t
			}
		}

		decisions[i] = &models.Decision{
			DecisionID:   decisionID,
			EventID:      d.EventID,
			TraceID:      d.TraceID,
			ItemID:       d.ItemID,
			Outcome:      d.Outcome,
			ReasonCode:   d.ReasonCode,
			ReasonDetail: d.ReasonDetail,
			Scores:       d.Scores,
			ItemSnapshot: d.ItemSnapshot,
			Timestamp:    timestamp,
		}
	}

	if err := h.store.BatchCreateDecisions(r.Context(), decisions); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to batch create decisions", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, models.APIResponse{
		Status: "created",
		Data:   map[string]int{"count": len(decisions)},
	})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message, details string) {
	log.Printf("ERROR [%d]: %s - %s", status, message, details)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(models.ErrorResponse{
		Error:   message,
		Details: details,
	})
}

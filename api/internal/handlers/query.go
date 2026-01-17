package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/xray-sdk/xray-api/internal/models"
	"github.com/xray-sdk/xray-api/internal/store"
)

// QueryHandler handles query endpoints
type QueryHandler struct {
	store store.Store
}

// NewQueryHandler creates a new query handler
func NewQueryHandler(s store.Store) *QueryHandler {
	return &QueryHandler{store: s}
}

// GetTrace handles GET /api/v1/traces/{traceId}
// @Summary Get a trace with all its events
// @Tags traces
// @Produce json
// @Param traceId path string true "Trace ID"
// @Success 200 {object} models.TraceWithEvents
// @Failure 404 {object} models.ErrorResponse
// @Router /traces/{traceId} [get]
func (h *QueryHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
	traceID := chi.URLParam(r, "traceId")

	trace, err := h.store.GetTraceWithEvents(r.Context(), traceID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get trace", err.Error())
		return
	}

	if trace == nil {
		respondError(w, http.StatusNotFound, "trace not found", "")
		return
	}

	respondJSON(w, http.StatusOK, trace)
}

// QueryTraces handles GET /api/v1/traces
// @Summary Query traces
// @Tags traces
// @Produce json
// @Param pipeline_id query string false "Filter by pipeline"
// @Param limit query int false "Max results"
// @Success 200 {object} models.PaginatedResponse
// @Router /traces [get]
func (h *QueryHandler) QueryTraces(w http.ResponseWriter, r *http.Request) {
	opts := &store.TraceQueryOpts{
		Limit: 100,
	}

	if pipelineID := r.URL.Query().Get("pipeline_id"); pipelineID != "" {
		opts.PipelineID = &pipelineID
	}

	if tagsStr := r.URL.Query().Get("tags"); tagsStr != "" {
		opts.Tags = strings.Split(tagsStr, ",")
	}

	// Parse metadata metadata filters: meta:key=value
	metadata := make(map[string]string)
	for key, values := range r.URL.Query() {
		if strings.HasPrefix(key, "meta:") && len(values) > 0 {
			metaKey := strings.TrimPrefix(key, "meta:")
			metadata[metaKey] = values[0]
		}
	}
	if len(metadata) > 0 {
		opts.Metadata = metadata
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			opts.Limit = limit
		}
	}

	page, err := h.store.QueryTraces(r.Context(), opts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query traces", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, models.PaginatedResponse{
		Results:    page.Traces,
		NextCursor: page.NextCursor,
		Count:      len(page.Traces),
	})
}

// GetEventsByTrace handles GET /api/v1/traces/{traceId}/events
// @Summary Get events for a trace
// @Tags events
// @Produce json
// @Param traceId path string true "Trace ID"
// @Success 200 {array} models.Event
// @Router /traces/{traceId}/events [get]
func (h *QueryHandler) GetEventsByTrace(w http.ResponseWriter, r *http.Request) {
	traceID := chi.URLParam(r, "traceId")

	events, err := h.store.GetEventsByTrace(r.Context(), traceID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get events", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, events)
}

// GetEvent handles GET /api/v1/traces/{traceId}/events/{eventId}
// @Summary Get a single event with summary
// @Tags events
// @Produce json
// @Param traceId path string true "Trace ID"
// @Param eventId path string true "Event ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} models.ErrorResponse
// @Router /traces/{traceId}/events/{eventId} [get]
func (h *QueryHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	traceID := chi.URLParam(r, "traceId")
	eventID := chi.URLParam(r, "eventId")

	event, err := h.store.GetEvent(r.Context(), traceID, eventID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get event", err.Error())
		return
	}

	if event == nil {
		respondError(w, http.StatusNotFound, "event not found", "")
		return
	}

	// Get associated decisions
	decisionsPage, err := h.store.GetDecisionsByEvent(r.Context(), eventID, nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get decisions", err.Error())
		return
	}

	// Compute decision summary with dynamic outcome aggregation
	totalDecisions := len(decisionsPage.Decisions)
	outcomeCounts := make(map[string]int)
	reasonCounts := make(map[string]int)

	for _, decision := range decisionsPage.Decisions {
		outcomeCounts[decision.Outcome]++
		if decision.ReasonCode != nil && *decision.ReasonCode != "" {
			reasonCounts[*decision.ReasonCode]++
		}
	}

	response := map[string]interface{}{
		"event": map[string]interface{}{
			"event_id":   event.EventID,
			"trace_id":   event.TraceID,
			"step_name":  event.StepName,
			"step_type":  event.StepType,
			"started_at": event.StartedAt,
			"ended_at":   event.EndedAt,
		},
		"metrics": event.Metrics, // Metrics is already a map
		"decisions": map[string]interface{}{
			"total":        totalDecisions,
			"outcomes":     outcomeCounts,
			"reason_codes": reasonCounts,
		},
	}

	respondJSON(w, http.StatusOK, response)
}

// QueryEvents handles GET /api/v1/query/events
// @Summary Query events across traces
// @Tags query
// @Produce json
// @Param step_type query string false "Filter by step type"
// @Param min_reduction_ratio query number false "Min reduction ratio"
// @Param limit query int false "Max results"
// @Success 200 {object} models.PaginatedResponse
// @Router /query/events [get]
func (h *QueryHandler) QueryEvents(w http.ResponseWriter, r *http.Request) {
	opts := &store.EventQueryOpts{
		Limit: 100,
	}

	if pipelineID := r.URL.Query().Get("pipeline_id"); pipelineID != "" {
		opts.PipelineID = &pipelineID
	}

	if stepType := r.URL.Query().Get("step_type"); stepType != "" {
		opts.StepType = &stepType
	}

	if minRedStr := r.URL.Query().Get("min_reduction_ratio"); minRedStr != "" {
		if minRed, err := strconv.ParseFloat(minRedStr, 64); err == nil {
			opts.MinReductionRatio = &minRed
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			opts.Limit = limit
		}
	}

	page, err := h.store.QueryEvents(r.Context(), opts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query events", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, models.PaginatedResponse{
		Results:    page.Events,
		NextCursor: page.NextCursor,
		Count:      len(page.Events),
	})
}

// GetDecisionsByEvent handles GET /api/v1/traces/{traceId}/events/{eventId}/decisions
// @Summary Get decisions for an event
// @Tags decisions
// @Produce json
// @Param traceId path string true "Trace ID"
// @Param eventId path string true "Event ID"
// @Param outcome query string false "Filter by outcome (e.g., 'accepted', 'rejected', 'escalated')"
// @Param reason_code query string false "Filter by reason code (e.g., 'COOLDOWN_ACTIVE', 'PRICE_TOO_HIGH')"
// @Param item_id query string false "Filter by specific item ID"
// @Param limit query int false "Max results (default 100)"
// @Success 200 {object} models.PaginatedResponse
// @Router /traces/{traceId}/events/{eventId}/decisions [get]
func (h *QueryHandler) GetDecisionsByEvent(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventId")

	opts := &store.DecisionQueryOpts{
		Limit: 100,
	}

	if outcome := r.URL.Query().Get("outcome"); outcome != "" {
		opts.Outcome = &outcome
	}

	if reasonCode := r.URL.Query().Get("reason_code"); reasonCode != "" {
		opts.ReasonCode = &reasonCode
	}

	if itemID := r.URL.Query().Get("item_id"); itemID != "" {
		opts.ItemID = &itemID
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			opts.Limit = limit
		}
	}

	page, err := h.store.GetDecisionsByEvent(r.Context(), eventID, opts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get decisions", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, models.PaginatedResponse{
		Results:    page.Decisions,
		NextCursor: page.NextCursor,
		Count:      len(page.Decisions),
	})
}

// Query handles POST /api/v1/query
// @Summary Advanced query
// @Tags query
// @Accept json
// @Produce json
// @Param query body models.QueryRequest true "Query parameters"
// @Success 200 {object} models.PaginatedResponse
// @Router /query [post]
func (h *QueryHandler) Query(w http.ResponseWriter, r *http.Request) {
	var req models.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Determine query type based on parameters
	if req.StepType != nil {
		// Event query
		opts := &store.EventQueryOpts{
			StepType:          req.StepType,
			MinReductionRatio: req.MinReductionRatio,
			Limit:             req.Limit,
		}
		if opts.Limit == 0 {
			opts.Limit = 100
		}

		page, err := h.store.QueryEvents(r.Context(), opts)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to query", err.Error())
			return
		}

		respondJSON(w, http.StatusOK, models.PaginatedResponse{
			Results:    page.Events,
			NextCursor: page.NextCursor,
			Count:      len(page.Events),
		})
		return
	}

	// Default to trace query
	opts := &store.TraceQueryOpts{
		PipelineID: req.PipelineID,
		Tags:       req.Tags,
		Metadata:   req.Metadata,
		Limit:      req.Limit,
	}
	if opts.Limit == 0 {
		opts.Limit = 100
	}

	page, err := h.store.QueryTraces(r.Context(), opts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, models.PaginatedResponse{
		Results:    page.Traces,
		NextCursor: page.NextCursor,
		Count:      len(page.Traces),
	})
}

// QueryDecisions handles GET /api/v1/query/decisions
// @Summary Query decisions across traces/events
// @Tags query
// @Produce json
// @Param pipeline_id query string false "Filter by pipeline ID"
// @Param step_name query string false "Filter by step name"
// @Param limit query int false "Max results"
// @Success 200 {object} models.PaginatedResponse
// @Router /query/decisions [get]
func (h *QueryHandler) QueryDecisions(w http.ResponseWriter, r *http.Request) {
	opts := &store.DecisionQueryOpts{
		Limit: 100,
	}

	if pipelineID := r.URL.Query().Get("pipeline_id"); pipelineID != "" {
		opts.PipelineID = &pipelineID
	}

	if stepName := r.URL.Query().Get("step_name"); stepName != "" {
		opts.StepName = &stepName
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			opts.Limit = limit
		}
	}

	page, err := h.store.QueryDecisions(r.Context(), opts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query decisions", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, models.PaginatedResponse{
		Results:    page.Decisions,
		NextCursor: page.NextCursor,
		Count:      len(page.Decisions),
	})
}

// Health handles GET /health
// @Summary Health check
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *QueryHandler) Health(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Ping(r.Context()); err != nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/auth"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/repository/clickhouse"
	"github.com/go-chi/chi/v5"
)

type TraceHandler struct {
	traceRepo *clickhouse.TraceRepository
}

func NewTraceHandler(traceRepo *clickhouse.TraceRepository) *TraceHandler {
	return &TraceHandler{traceRepo: traceRepo}
}

func (h *TraceHandler) SearchTraces(w http.ResponseWriter, r *http.Request) {
	orgID := auth.GetOrgID(r.Context())
	if orgID == "" {
		WriteError(w, http.StatusUnauthorized, "organization context missing")
		return
	}

	q := r.URL.Query()
	service := q.Get("service")
	status := q.Get("status")

	minDur, _ := strconv.Atoi(q.Get("min_duration_ms"))
	maxDur, _ := strconv.Atoi(q.Get("max_duration_ms"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	if limit <= 0 || limit > 200 {
		limit = 50
	}

	filters := domain.TraceFilter{
		Service:       service,
		Status:        status,
		MinDurationMs: minDur,
		MaxDurationMs: maxDur,
		Limit:         limit,
		Offset:        offset,
	}

	if fromStr := q.Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filters.From = t
		}
	}
	if toStr := q.Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			filters.To = t
		}
	}

	traces, err := h.traceRepo.SearchTraces(r.Context(), orgID, filters)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to search traces: "+err.Error())
		return
	}

	if traces == nil {
		traces = []domain.TraceRow{}
	}

	WriteJSON(w, http.StatusOK, traces)
}

func (h *TraceHandler) GetTraceByID(w http.ResponseWriter, r *http.Request) {
	orgID := auth.GetOrgID(r.Context())
	if orgID == "" {
		WriteError(w, http.StatusUnauthorized, "organization context missing")
		return
	}

	traceID := chi.URLParam(r, "trace_id")
	if traceID == "" {
		traceID = chi.URLParam(r, "id")
	}
	if traceID == "" {
		WriteError(w, http.StatusBadRequest, "trace id is required")
		return
	}

	trace, err := h.traceRepo.GetTraceWithSpans(r.Context(), orgID, traceID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to get trace: "+err.Error())
		return
	}

	if trace == nil {
		WriteError(w, http.StatusNotFound, "trace not found")
		return
	}

	WriteJSON(w, http.StatusOK, trace)
}

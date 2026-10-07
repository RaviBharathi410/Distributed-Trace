package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"context"

	"github.com/RaviBharathi410/distributedtrace/internal/auth"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/repository/clickhouse"
	"github.com/go-chi/chi/v5"
)

// AnomalyExplainer defines the interface for generating grounded incident explanations.
type AnomalyExplainer interface {
	Explain(ctx context.Context, orgID string, anomaly *domain.Anomaly, forceRefresh bool) (*domain.IncidentExplanation, error)
}

type AnomalyHandler struct {
	anomalyRepo *clickhouse.AnomalyRepository
	explainer   AnomalyExplainer
}

func NewAnomalyHandler(anomalyRepo *clickhouse.AnomalyRepository) *AnomalyHandler {
	return &AnomalyHandler{anomalyRepo: anomalyRepo}
}

func NewAnomalyHandlerWithExplainer(anomalyRepo *clickhouse.AnomalyRepository, explainer AnomalyExplainer) *AnomalyHandler {
	return &AnomalyHandler{
		anomalyRepo: anomalyRepo,
		explainer:   explainer,
	}
}

func (h *AnomalyHandler) SetExplainer(explainer AnomalyExplainer) {
	h.explainer = explainer
}

func (h *AnomalyHandler) ListAnomalies(w http.ResponseWriter, r *http.Request) {
	orgID := auth.GetOrgID(r.Context())
	if orgID == "" {
		WriteError(w, http.StatusUnauthorized, "organization context missing")
		return
	}

	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	filters := domain.AnomalyFilter{
		Severity: q.Get("severity"),
		Service:  q.Get("service"),
		Status:   q.Get("status"),
		Limit:    limit,
		Offset:   offset,
	}

	anomalies, err := h.anomalyRepo.List(r.Context(), orgID, filters)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to list anomalies: "+err.Error())
		return
	}

	if anomalies == nil {
		anomalies = []domain.Anomaly{}
	}

	stats, _ := h.anomalyRepo.GetStats(r.Context(), orgID)
	if stats == nil {
		stats = &domain.AnomalyStats{}
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"anomalies": anomalies,
		"stats":     stats,
	})
}

func (h *AnomalyHandler) GetAnomalyByID(w http.ResponseWriter, r *http.Request) {
	orgID := auth.GetOrgID(r.Context())
	if orgID == "" {
		WriteError(w, http.StatusUnauthorized, "organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	anomaly, err := h.anomalyRepo.GetByID(r.Context(), orgID, id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to get anomaly: "+err.Error())
		return
	}

	if anomaly == nil {
		WriteError(w, http.StatusNotFound, "anomaly not found")
		return
	}

	WriteJSON(w, http.StatusOK, anomaly)
}

type UpdateAnomalyStatusRequest struct {
	Status string `json:"status"` // "open", "investigating", "resolved"
}

func (h *AnomalyHandler) UpdateAnomalyStatus(w http.ResponseWriter, r *http.Request) {
	orgID := auth.GetOrgID(r.Context())
	if orgID == "" {
		WriteError(w, http.StatusUnauthorized, "organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	var req UpdateAnomalyStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if req.Status != "open" && req.Status != "investigating" && req.Status != "resolved" {
		WriteError(w, http.StatusBadRequest, "status must be 'open', 'investigating', or 'resolved'")
		return
	}

	if err := h.anomalyRepo.UpdateStatus(r.Context(), orgID, id, req.Status); err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to update status: "+err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"id":     id,
		"status": req.Status,
	})
}

// ExplainAnomaly triggers on-demand, read-only root cause explanation for an anomaly.
func (h *AnomalyHandler) ExplainAnomaly(w http.ResponseWriter, r *http.Request) {
	orgID := auth.GetOrgID(r.Context())
	if orgID == "" {
		WriteError(w, http.StatusUnauthorized, "organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		WriteError(w, http.StatusBadRequest, "anomaly id required")
		return
	}

	// Fetch anomaly with strict tenant isolation
	anomaly, err := h.anomalyRepo.GetByID(r.Context(), orgID, id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to get anomaly: "+err.Error())
		return
	}
	if anomaly == nil {
		WriteError(w, http.StatusNotFound, "anomaly not found")
		return
	}

	// Parse optional request body for force_refresh
	var req domain.ExplainAnomalyRequest
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	if h.explainer == nil {
		WriteError(w, http.StatusServiceUnavailable, "incident explainer engine is not configured")
		return
	}

	explanation, err := h.explainer.Explain(r.Context(), orgID, anomaly, req.ForceRefresh)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to explain incident: "+err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, explanation)
}


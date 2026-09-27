package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/RaviBharathi410/distributedtrace/internal/auth"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/repository/clickhouse"
	"github.com/go-chi/chi/v5"
)

type AnomalyHandler struct {
	anomalyRepo *clickhouse.AnomalyRepository
}

func NewAnomalyHandler(anomalyRepo *clickhouse.AnomalyRepository) *AnomalyHandler {
	return &AnomalyHandler{anomalyRepo: anomalyRepo}
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

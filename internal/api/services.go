package api

import (
	"net/http"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/auth"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/repository/clickhouse"
	"github.com/go-chi/chi/v5"
)

type ServiceHandler struct {
	serviceRepo *clickhouse.ServiceRepository
}

func NewServiceHandler(serviceRepo *clickhouse.ServiceRepository) *ServiceHandler {
	return &ServiceHandler{serviceRepo: serviceRepo}
}

func (h *ServiceHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	orgID := auth.GetOrgID(r.Context())
	if orgID == "" {
		WriteError(w, http.StatusUnauthorized, "organization context missing")
		return
	}

	services, err := h.serviceRepo.ListServices(r.Context(), orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to list services: "+err.Error())
		return
	}

	if services == nil {
		services = []string{}
	}

	WriteJSON(w, http.StatusOK, services)
}

func (h *ServiceHandler) GetServiceGraph(w http.ResponseWriter, r *http.Request) {
	orgID := auth.GetOrgID(r.Context())
	if orgID == "" {
		WriteError(w, http.StatusUnauthorized, "organization context missing")
		return
	}

	to := time.Now().UTC()
	from := to.Add(-1 * time.Hour) // default last 1 hour

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = t
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = t
		}
	}

	graph, err := h.serviceRepo.GetServiceGraph(r.Context(), orgID, from, to)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to compute service graph: "+err.Error())
		return
	}

	if graph == nil {
		graph = &domain.ServiceGraph{
			Nodes: []domain.ServiceNode{},
			Edges: []domain.ServiceEdge{},
		}
	}

	WriteJSON(w, http.StatusOK, graph)
}

func (h *ServiceHandler) GetServiceStats(w http.ResponseWriter, r *http.Request) {
	orgID := auth.GetOrgID(r.Context())
	if orgID == "" {
		WriteError(w, http.StatusUnauthorized, "organization context missing")
		return
	}

	serviceName := chi.URLParam(r, "service")
	if serviceName == "" {
		WriteError(w, http.StatusBadRequest, "service name is required")
		return
	}

	to := time.Now().UTC()
	from := to.Add(-1 * time.Hour)

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = t
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = t
		}
	}

	stats, err := h.serviceRepo.GetServiceStats(r.Context(), orgID, serviceName, from, to)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to get service stats: "+err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, stats)
}

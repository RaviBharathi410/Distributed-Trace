package api

import (
	"net/http"
	"strconv"

	"github.com/RaviBharathi410/distributedtrace/internal/auth"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/repository/postgres"
)

type CostHandler struct {
	costRepo *postgres.CostRepository
}

func NewCostHandler(costRepo *postgres.CostRepository) *CostHandler {
	return &CostHandler{costRepo: costRepo}
}

// GetCostSummary returns the organization's cost proration, incident LLM spend, and ceiling status.
func (h *CostHandler) GetCostSummary(w http.ResponseWriter, r *http.Request) {
	orgID := auth.GetOrgID(r.Context())
	if orgID == "" {
		WriteError(w, http.StatusUnauthorized, "organization context missing")
		return
	}

	summary, err := h.costRepo.GetCostSummary(r.Context(), orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to get cost summary: "+err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, summary)
}

// GetCostBreakdown returns the per-incident cost ledger for the caller's organization.
func (h *CostHandler) GetCostBreakdown(w http.ResponseWriter, r *http.Request) {
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
	if offset < 0 {
		offset = 0
	}

	items, err := h.costRepo.GetCostBreakdown(r.Context(), orgID, limit, offset)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to get cost breakdown: "+err.Error())
		return
	}

	if items == nil {
		items = []domain.CostBreakdownItem{}
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"limit":  limit,
		"offset": offset,
	})
}

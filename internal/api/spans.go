package api

import (
	"encoding/json"
	"net/http"

	"github.com/RaviBharathi410/distributedtrace/internal/auth"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/ingest"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
	"github.com/RaviBharathi410/distributedtrace/internal/repository/clickhouse"
	"go.uber.org/zap"
)

type SpanHandler struct {
	traceRepo *clickhouse.TraceRepository
	producer  *ingest.SpanProducer
}

func NewSpanHandler(traceRepo *clickhouse.TraceRepository, producer *ingest.SpanProducer) *SpanHandler {
	return &SpanHandler{
		traceRepo: traceRepo,
		producer:  producer,
	}
}

type IngestResponse struct {
	Ingested int    `json:"ingested"`
	Status   string `json:"status"`
}

func (h *SpanHandler) IngestSpans(w http.ResponseWriter, r *http.Request) {
	orgID := auth.GetOrgID(r.Context())
	if orgID == "" {
		WriteError(w, http.StatusUnauthorized, "organization context missing")
		return
	}

	var spans []domain.Span
	if err := json.NewDecoder(r.Body).Decode(&spans); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	if len(spans) == 0 {
		WriteError(w, http.StatusBadRequest, "no spans provided in batch")
		return
	}

	// 1. Validate incoming telemetry
	if err := ingest.ValidateSpans(spans); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 2. Enrich with OrgID and normalized names
	enrichedSpans := ingest.EnrichSpans(spans, orgID)

	// 3. Write directly to ClickHouse
	if err := h.traceRepo.InsertSpansBatch(r.Context(), enrichedSpans); err != nil {
		observability.Log.Error("Failed to insert spans into ClickHouse", zap.Error(err), zap.String("org_id", orgID))
		WriteError(w, http.StatusInternalServerError, "failed to persist spans: "+err.Error())
		return
	}

	// 4. Also publish to Kafka if producer is active
	if h.producer != nil {
		go func(spansToSend []domain.Span) {
			if err := h.producer.PublishSpans(r.Context(), spansToSend); err != nil {
				observability.Log.Warn("Async publish to Kafka failed", zap.Error(err))
			}
		}(enrichedSpans)
	}

	observability.SpansIngested.WithLabelValues(orgID, "ok").Add(float64(len(spans)))

	WriteJSON(w, http.StatusAccepted, IngestResponse{
		Ingested: len(enrichedSpans),
		Status:   "accepted",
	})
}

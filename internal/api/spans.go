package api

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/RaviBharathi410/distributedtrace/internal/auth"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/ingest"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
	"github.com/RaviBharathi410/distributedtrace/internal/repository/clickhouse"
	"go.uber.org/zap"
)

type SpanPublisher interface {
	PublishSpans(ctx context.Context, spans []domain.Span) error
}

type SpanHandler struct {
	traceRepo *clickhouse.TraceRepository
	producer  SpanPublisher
}

func NewSpanHandler(traceRepo *clickhouse.TraceRepository, producer SpanPublisher) *SpanHandler {
	return &SpanHandler{
		traceRepo: traceRepo,
		producer:  producer,
	}
}

const (
	// MaxIngestBodyBytes limits the maximum decompressed ingest body size (4 MB).
	// Protects against gzip bombs and memory exhaustion attacks.
	MaxIngestBodyBytes int64 = 4 * 1024 * 1024

	// MaxSpansPerBatch limits the maximum number of spans accepted per batch.
	MaxSpansPerBatch = 1000
)

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

	// 1. Guard against oversized payloads: limit body stream to MaxIngestBodyBytes + 1
	limitedReader := io.LimitReader(r.Body, MaxIngestBodyBytes+1)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "failed to read request body: "+err.Error())
		return
	}
	if int64(len(bodyBytes)) > MaxIngestBodyBytes {
		WriteError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("payload exceeds maximum allowed size of %d bytes", MaxIngestBodyBytes))
		return
	}

	var spans []domain.Span
	if err := json.Unmarshal(bodyBytes, &spans); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	if len(spans) == 0 {
		WriteError(w, http.StatusBadRequest, "no spans provided in batch")
		return
	}

	if len(spans) > MaxSpansPerBatch {
		WriteError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("batch size %d exceeds maximum limit of %d spans", len(spans), MaxSpansPerBatch))
		return
	}

	// 2. Validate incoming telemetry
	if err := ingest.ValidateSpans(spans); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 3. Enrich with OrgID and normalized names
	enrichedSpans := ingest.EnrichSpans(spans, orgID)

	// 4. Decoupled Pipeline (Decision #19):
	// If Kafka producer is configured, publish exclusively to Kafka and let batch consumer persist to ClickHouse.
	// If producer is nil (dev/test fallback), insert directly to ClickHouse.
	if h.producer != nil {
		if err := h.producer.PublishSpans(r.Context(), enrichedSpans); err != nil {
			observability.Log.Error("Failed to publish spans to Kafka", zap.Error(err), zap.String("org_id", orgID))
			WriteError(w, http.StatusInternalServerError, "failed to queue spans: "+err.Error())
			return
		}
	} else {
		if err := h.traceRepo.InsertSpansBatch(r.Context(), enrichedSpans); err != nil {
			observability.Log.Error("Failed to insert spans into ClickHouse", zap.Error(err), zap.String("org_id", orgID))
			WriteError(w, http.StatusInternalServerError, "failed to persist spans: "+err.Error())
			return
		}
	}

	observability.SpansIngested.WithLabelValues(orgID, "ok").Add(float64(len(spans)))

	WriteJSON(w, http.StatusAccepted, IngestResponse{
		Ingested: len(enrichedSpans),
		Status:   "accepted",
	})
}

// IngestOTLP accepts standard OpenTelemetry Protocol HTTP (OTLP/HTTP) payloads (POST /v1/traces)
// Supports both Protobuf (application/x-protobuf) and JSON (application/json), with optional gzip compression.
func (h *SpanHandler) IngestOTLP(w http.ResponseWriter, r *http.Request) {
	orgID := auth.GetOrgID(r.Context())
	if orgID == "" {
		WriteError(w, http.StatusUnauthorized, "organization context missing")
		return
	}

	// 1. Decompress if body is gzip encoded (Content-Encoding: gzip)
	var reader io.Reader = r.Body
	if strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
		gzReader, err := gzip.NewReader(r.Body)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid gzip body: "+err.Error())
			return
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// 2. Protect against zip bombs and memory exhaustion: limit stream to MaxIngestBodyBytes + 1
	limitedReader := io.LimitReader(reader, MaxIngestBodyBytes+1)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "failed to read request body: "+err.Error())
		return
	}
	if int64(len(bodyBytes)) > MaxIngestBodyBytes {
		WriteError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("decompressed payload exceeds maximum allowed size of %d bytes", MaxIngestBodyBytes))
		return
	}

	// 3. Decode based on Content-Type
	contentType := r.Header.Get("Content-Type")
	spans, err := ingest.ParseOTLPTraces(bodyBytes, contentType)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid OTLP payload: "+err.Error())
		return
	}

	if len(spans) == 0 {
		h.respondOTLPSuccess(w, r)
		return
	}

	// 4. Cap batch size (413 if batch exceeds limit)
	if len(spans) > MaxSpansPerBatch {
		WriteError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("batch size %d exceeds maximum limit of %d spans", len(spans), MaxSpansPerBatch))
		return
	}

	// 5. Validate incoming telemetry
	if err := ingest.ValidateSpans(spans); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 6. Enrich with OrgID and normalized names (enforcing tenant isolation and clamping org_id tags)
	enrichedSpans := ingest.EnrichSpans(spans, orgID)

	// 7. Decoupled Pipeline (Decision #19):
	if h.producer != nil {
		if err := h.producer.PublishSpans(r.Context(), enrichedSpans); err != nil {
			observability.Log.Error("Failed to publish OTLP spans to Kafka", zap.Error(err), zap.String("org_id", orgID))
			WriteError(w, http.StatusInternalServerError, "failed to queue spans: "+err.Error())
			return
		}
	} else {
		if err := h.traceRepo.InsertSpansBatch(r.Context(), enrichedSpans); err != nil {
			observability.Log.Error("Failed to insert OTLP spans into ClickHouse", zap.Error(err), zap.String("org_id", orgID))
			WriteError(w, http.StatusInternalServerError, "failed to persist spans: "+err.Error())
			return
		}
	}

	observability.SpansIngested.WithLabelValues(orgID, "ok").Add(float64(len(spans)))

	h.respondOTLPSuccess(w, r)
}

func (h *SpanHandler) respondOTLPSuccess(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	contentType := r.Header.Get("Content-Type")

	// Standard OTel SDKs and Collectors requesting protobuf expect binary application/x-protobuf response
	if strings.Contains(accept, "application/x-protobuf") || strings.Contains(contentType, "application/x-protobuf") {
		respBytes, err := ingest.BuildOTLPProtoSuccessResponse()
		if err == nil {
			w.Header().Set("Content-Type", "application/x-protobuf")
			w.WriteHeader(http.StatusOK)
			w.Write(respBytes)
			return
		}
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"partialSuccess": map[string]any{},
	})
}


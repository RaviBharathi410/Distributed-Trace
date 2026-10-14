package api

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/auth"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	chRepo "github.com/RaviBharathi410/distributedtrace/internal/repository/clickhouse"
	"github.com/go-chi/chi/v5"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

type mockSpanPublisher struct {
	publishedSpans []domain.Span
	publishErr     error
}

func (m *mockSpanPublisher) PublishSpans(ctx context.Context, spans []domain.Span) error {
	if m.publishErr != nil {
		return m.publishErr
	}
	m.publishedSpans = append(m.publishedSpans, spans...)
	return nil
}

func TestSpanHandler_DecoupledPipeline(t *testing.T) {
	now := time.Now()
	validSpan := domain.Span{
		TraceID:       "trace-producer-1",
		SpanID:        "span-1",
		ServiceName:   "order-service",
		OperationName: "CreateOrder",
		DurationMs:    150,
		StatusCode:    1,
		StartTime:     now,
	}

	t.Run("Publishes to Kafka when producer is configured (does not call ClickHouse)", func(t *testing.T) {
		mockConn := &apiMockConn{}
		traceRepo := chRepo.NewTraceRepository(mockConn)
		mockPub := &mockSpanPublisher{}
		handler := NewSpanHandler(traceRepo, mockPub)

		body, _ := json.Marshal([]domain.Span{validSpan})
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		req := httptest.NewRequest(http.MethodPost, "/spans", bytes.NewReader(body)).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.IngestSpans(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Fatalf("expected 202 Accepted, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Assert publisher received the enriched span
		if len(mockPub.publishedSpans) != 1 {
			t.Fatalf("expected 1 span published to Kafka, got %d", len(mockPub.publishedSpans))
		}
		if mockPub.publishedSpans[0].OrgID != TestOrgA {
			t.Errorf("expected published span org_id to be %q, got %q", TestOrgA, mockPub.publishedSpans[0].OrgID)
		}

		// Assert ClickHouse was NOT called (no rows appended to batch)
		if len(mockConn.lastQuery) > 0 {
			t.Errorf("expected ClickHouse not to be queried, but got lastQuery: %s", mockConn.lastQuery)
		}
	})

	t.Run("Returns 500 when Kafka producer fails", func(t *testing.T) {
		mockConn := &apiMockConn{}
		traceRepo := chRepo.NewTraceRepository(mockConn)
		mockPub := &mockSpanPublisher{
			publishErr: errors.New("kafka broker connection refused"),
		}
		handler := NewSpanHandler(traceRepo, mockPub)

		body, _ := json.Marshal([]domain.Span{validSpan})
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		req := httptest.NewRequest(http.MethodPost, "/spans", bytes.NewReader(body)).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.IngestSpans(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 Internal Server Error, got %d", rec.Code)
		}
	})

	t.Run("Falls back to direct ClickHouse insert when producer is nil", func(t *testing.T) {
		mockConn := &apiMockConn{}
		traceRepo := chRepo.NewTraceRepository(mockConn)
		handler := NewSpanHandler(traceRepo, nil)

		body, _ := json.Marshal([]domain.Span{validSpan})
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		req := httptest.NewRequest(http.MethodPost, "/spans", bytes.NewReader(body)).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.IngestSpans(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Fatalf("expected 202 Accepted, got %d", rec.Code)
		}
	})
}

func TestSpanHandler_IngestPayloadSizeLimits(t *testing.T) {
	mockConn := &apiMockConn{}
	traceRepo := chRepo.NewTraceRepository(mockConn)
	handler := NewSpanHandler(traceRepo, nil)
	ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)

	t.Run("IngestSpans rejects payload exceeding MaxIngestBodyBytes with 413", func(t *testing.T) {
		oversized := bytes.Repeat([]byte(" "), int(MaxIngestBodyBytes)+1024)
		req := httptest.NewRequest(http.MethodPost, "/spans", bytes.NewReader(oversized)).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.IngestSpans(rec, req)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected 413 Request Entity Too Large, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("IngestSpans rejects batch exceeding MaxSpansPerBatch with 413", func(t *testing.T) {
		now := time.Now()
		oversizedBatch := make([]domain.Span, MaxSpansPerBatch+1)
		for i := range oversizedBatch {
			oversizedBatch[i] = domain.Span{
				TraceID:       "trace-1",
				SpanID:        fmt.Sprintf("span-%d", i),
				ServiceName:   "svc",
				OperationName: "op",
				DurationMs:    10,
				StatusCode:    1,
				StartTime:     now,
			}
		}

		body, _ := json.Marshal(oversizedBatch)
		req := httptest.NewRequest(http.MethodPost, "/spans", bytes.NewReader(body)).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.IngestSpans(rec, req)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected 413 Request Entity Too Large for 1001 spans, got %d", rec.Code)
		}
	})

	t.Run("IngestOTLP rejects uncompressed payload exceeding MaxIngestBodyBytes with 413", func(t *testing.T) {
		oversized := bytes.Repeat([]byte(" "), int(MaxIngestBodyBytes)+1024)
		req := httptest.NewRequest(http.MethodPost, "/v1/traces", bytes.NewReader(oversized)).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.IngestOTLP(rec, req)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected 413 Request Entity Too Large, got %d", rec.Code)
		}
	})

	t.Run("IngestOTLP neutralizes gzip bomb exceeding MaxIngestBodyBytes with 413", func(t *testing.T) {
		var gzBuf bytes.Buffer
		gzWriter := gzip.NewWriter(&gzBuf)
		uncompressedOversized := bytes.Repeat([]byte("A"), int(MaxIngestBodyBytes)+1024)
		if _, err := gzWriter.Write(uncompressedOversized); err != nil {
			t.Fatalf("failed to write gzip payload: %v", err)
		}
		if err := gzWriter.Close(); err != nil {
			t.Fatalf("failed to close gzip: %v", err)
		}

		t.Logf("Gzip bomb payload size: %d bytes (decompresses to %d bytes)", gzBuf.Len(), len(uncompressedOversized))

		req := httptest.NewRequest(http.MethodPost, "/v1/traces", &gzBuf).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		rec := httptest.NewRecorder()

		handler.IngestOTLP(rec, req)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected 413 Request Entity Too Large for gzip bomb, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("IngestOTLP rejects batch exceeding MaxSpansPerBatch with 413", func(t *testing.T) {
		protoSpans := make([]*tracepb.Span, MaxSpansPerBatch+1)
		for i := range protoSpans {
			protoSpans[i] = &tracepb.Span{
				TraceId:           []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				SpanId:            []byte{1, 2, 3, 4, 5, 6, 7, 8},
				Name:              "op",
				StartTimeUnixNano: uint64(time.Now().UnixNano()),
				EndTimeUnixNano:   uint64(time.Now().Add(10 * time.Millisecond).UnixNano()),
			}
		}

		protoReq := &coltracepb.ExportTraceServiceRequest{
			ResourceSpans: []*tracepb.ResourceSpans{
				{
					ScopeSpans: []*tracepb.ScopeSpans{
						{Spans: protoSpans},
					},
				},
			},
		}

		body, _ := proto.Marshal(protoReq)
		req := httptest.NewRequest(http.MethodPost, "/v1/traces", bytes.NewReader(body)).WithContext(ctx)
		req.Header.Set("Content-Type", "application/x-protobuf")
		rec := httptest.NewRecorder()

		handler.IngestOTLP(rec, req)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected 413 Request Entity Too Large for >1000 OTLP spans, got %d", rec.Code)
		}
	})
}

func TestOTLPEndpoint_RouteAuthMount_Strict401(t *testing.T) {
	svc, err := auth.NewTokenServiceFromPaths("", "", "test-issuer", "test")
	if err != nil {
		t.Fatalf("failed to create token service: %v", err)
	}

	mockConn := &apiMockConn{}
	traceRepo := chRepo.NewTraceRepository(mockConn)
	spanHandler := NewSpanHandler(traceRepo, nil)

	r := chi.NewRouter()
	// Mount routes identically to cmd/server/main.go
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireIngestAuth(nil, svc))
		r.Post("/v1/traces", spanHandler.IngestOTLP)
		r.Post("/v1/traces/", spanHandler.IngestOTLP)
	})
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/traces/otlp", func(r chi.Router) {
			r.Use(auth.RequireIngestAuth(nil, svc))
			r.Post("/", spanHandler.IngestOTLP)
		})
	})

	endpoints := []string{"/v1/traces", "/v1/traces/", "/api/v1/traces/otlp"}
	for _, ep := range endpoints {
		t.Run("No auth header to "+ep+" returns 401", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, ep, bytes.NewReader([]byte("{}")))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("SECURITY VIOLATION: expected 401 on unauthenticated %s, got %d", ep, rec.Code)
			}
		})

		t.Run("Invalid API key to "+ep+" returns 401", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, ep, bytes.NewReader([]byte("{}")))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-API-Key", "dt_live_invalid_key_value")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("SECURITY VIOLATION: expected 401 on invalid key for %s, got %d", ep, rec.Code)
			}
		})
	}

	t.Run("Valid JWT to /v1/traces is accepted and authenticated", func(t *testing.T) {
		token, err := svc.GenerateToken("user-1", "org-allowed", "admin", 1*time.Hour)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/traces", bytes.NewReader([]byte(`{"resourceSpans":[]}`)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK with valid credentials, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestSpanHandler_IngestOTLP_8192Batch_HandlerToClickHouse(t *testing.T) {
	mockConn := &apiMockConn{}
	traceRepo := chRepo.NewTraceRepository(mockConn)
	handler := NewSpanHandler(traceRepo, nil)

	const batchSize = 8192
	protoSpans := make([]*tracepb.Span, batchSize)
	now := time.Now()

	for i := 0; i < batchSize; i++ {
		traceID := make([]byte, 16)
		traceID[0] = 0xAA
		traceID[15] = byte(i % 256)

		spanID := make([]byte, 8)
		spanID[0] = 0xBB
		spanID[6] = byte((i >> 8) % 256)
		spanID[7] = byte(i % 256)

		protoSpans[i] = &tracepb.Span{
			TraceId:           traceID,
			SpanId:            spanID,
			Name:              fmt.Sprintf("operation-%d", i%15),
			Kind:              tracepb.Span_SPAN_KIND_SERVER,
			StartTimeUnixNano: uint64(now.UnixNano()),
			EndTimeUnixNano:   uint64(now.Add(25 * time.Millisecond).UnixNano()),
			Attributes: []*commonpb.KeyValue{
				{
					Key:   "http.status_code",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 200}},
				},
				{
					Key:   "http.method",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "POST"}},
				},
			},
			Status: &tracepb.Status{
				Code: tracepb.Status_STATUS_CODE_OK,
			},
		}
	}

	protoReq := &coltracepb.ExportTraceServiceRequest{
		ResourceSpans: []*tracepb.ResourceSpans{
			{
				Resource: &resourcepb.Resource{
					Attributes: []*commonpb.KeyValue{
						{
							Key:   "service.name",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "collector-batch-service"}},
						},
					},
				},
				ScopeSpans: []*tracepb.ScopeSpans{
					{
						Spans: protoSpans,
					},
				},
			},
		},
	}

	rawBytes, err := proto.Marshal(protoReq)
	if err != nil {
		t.Fatalf("failed to marshal 8,192 protobuf spans: %v", err)
	}

	var gzBuf bytes.Buffer
	gzWriter := gzip.NewWriter(&gzBuf)
	if _, err := gzWriter.Write(rawBytes); err != nil {
		t.Fatalf("failed to gzip compress 8,192 spans: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	t.Logf("8,192 OTLP Collector batch wire size: %d bytes raw, %d bytes gzip compressed", len(rawBytes), gzBuf.Len())

	ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
	req := httptest.NewRequest(http.MethodPost, "/v1/traces", &gzBuf).WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.IngestOTLP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for 8,192-span Collector batch, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Verify ClickHouse batch persistence
	if len(mockConn.batches) == 0 {
		t.Fatalf("expected ClickHouse batch to be prepared for 8,192 spans")
	}

	totalPersisted := 0
	for _, b := range mockConn.batches {
		totalPersisted += len(b.appendedRows)
		for _, row := range b.appendedRows {
			orgID, ok := row[0].(string)
			if !ok || orgID != TestOrgA {
				t.Fatalf("tenant isolation violation: expected org_id %q, got %v", TestOrgA, row[0])
			}
		}
	}

	if totalPersisted != batchSize {
		t.Fatalf("expected exactly %d spans persisted to ClickHouse, got %d", batchSize, totalPersisted)
	}

	t.Logf("✔ Successfully ingested and persisted full 8,192-span OpenTelemetry Collector batch from handler to ClickHouse with strict tenant isolation!")
}



package api

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
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

const (
	TestOrgA = "org-alpha-111"
	TestOrgB = "org-beta-222"
)

// Mock ClickHouse Connection for API tests
type apiMockConn struct {
	lastQuery string
	lastArgs  []any
	batches   []*apiMockBatch
}

func (m *apiMockConn) Contributors() []string                          { return nil }
func (m *apiMockConn) ServerVersion() (*driver.ServerVersion, error)  { return nil, nil }
func (m *apiMockConn) Select(ctx context.Context, dest any, query string, args ...any) error {
	return nil
}
func (m *apiMockConn) Ping(ctx context.Context) error                 { return nil }
func (m *apiMockConn) Stats() driver.Stats                             { return driver.Stats{} }
func (m *apiMockConn) Close() error                                    { return nil }
func (m *apiMockConn) AsyncInsert(ctx context.Context, query string, wait bool, args ...any) error {
	return nil
}
func (m *apiMockConn) PrepareBatch(ctx context.Context, query string, opts ...driver.PrepareBatchOption) (driver.Batch, error) {
	b := &apiMockBatch{conn: m}
	m.batches = append(m.batches, b)
	return b, nil
}
func (m *apiMockConn) Exec(ctx context.Context, query string, args ...any) error {
	m.lastQuery = query
	m.lastArgs = args
	return nil
}
func (m *apiMockConn) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	m.lastQuery = query
	m.lastArgs = args
	return &apiMockRow{}
}
func (m *apiMockConn) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	m.lastQuery = query
	m.lastArgs = args
	return &apiMockRows{}, nil
}

type apiMockRow struct{}

func (r *apiMockRow) Scan(dest ...any) error      { return nil }
func (r *apiMockRow) ScanStruct(dest any) error { return nil }
func (r *apiMockRow) Err() error                { return nil }

type apiMockRows struct{}

func (r *apiMockRows) Next() bool                      { return false }
func (r *apiMockRows) Scan(dest ...any) error          { return nil }
func (r *apiMockRows) ScanStruct(dest any) error       { return nil }
func (r *apiMockRows) ColumnTypes() []driver.ColumnType { return nil }
func (r *apiMockRows) Totals(dest ...any) error        { return nil }
func (r *apiMockRows) Columns() []string               { return nil }
func (r *apiMockRows) Close() error                    { return nil }
func (r *apiMockRows) Err() error                      { return nil }

type apiMockBatch struct {
	conn         *apiMockConn
	appendedRows [][]any
}

func (b *apiMockBatch) Abort() error               { return nil }
func (b *apiMockBatch) Append(v ...any) error       { b.appendedRows = append(b.appendedRows, v); return nil }
func (b *apiMockBatch) AppendStruct(v any) error    { return nil }
func (b *apiMockBatch) Column(int) driver.BatchColumn { return nil }
func (b *apiMockBatch) Flush() error                { return nil }
func (b *apiMockBatch) Send() error                 { return nil }
func (b *apiMockBatch) IsSent() bool                { return true }
func (b *apiMockBatch) Rows() int                   { return len(b.appendedRows) }

func TestServicesHandler_TenantIsolation(t *testing.T) {
	mockConn := &apiMockConn{}
	serviceRepo := chRepo.NewServiceRepository(mockConn)
	handler := NewServiceHandler(serviceRepo)

	t.Run("Rejects request when organization context is missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/services", nil)
		rec := httptest.NewRecorder()

		handler.ListServices(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Passes auth context org_id to ListServices", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		req := httptest.NewRequest(http.MethodGet, "/services", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.ListServices(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", rec.Code)
		}

		if len(mockConn.lastArgs) == 0 || mockConn.lastArgs[0] != TestOrgA {
			t.Errorf("expected org_id arg to be %q, got %v", TestOrgA, mockConn.lastArgs)
		}
	})

	t.Run("Passes auth context org_id to GetServiceGraph", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgB)
		req := httptest.NewRequest(http.MethodGet, "/services/graph", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.GetServiceGraph(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", rec.Code)
		}

		// Verify that TestOrgB was passed as argument in ClickHouse query
		found := false
		for _, arg := range mockConn.lastArgs {
			if s, ok := arg.(string); ok && s == TestOrgB {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %q in query args, got %v", TestOrgB, mockConn.lastArgs)
		}
	})
}

func TestTracesHandler_TenantIsolation(t *testing.T) {
	mockConn := &apiMockConn{}
	traceRepo := chRepo.NewTraceRepository(mockConn)
	handler := NewTraceHandler(traceRepo)

	t.Run("Rejects SearchTraces without org context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/traces", nil)
		rec := httptest.NewRecorder()

		handler.SearchTraces(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Passes auth context org_id to SearchTraces", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		req := httptest.NewRequest(http.MethodGet, "/traces", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.SearchTraces(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", rec.Code)
		}

		if len(mockConn.lastArgs) == 0 || mockConn.lastArgs[0] != TestOrgA {
			t.Errorf("expected first query arg to be %q, got %v", TestOrgA, mockConn.lastArgs)
		}
	})

	t.Run("GetTraceByID extracts trace_id URL param and binds org_id", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		r := chi.NewRouter()
		r.Get("/traces/{trace_id}", handler.GetTraceByID)

		req := httptest.NewRequest(http.MethodGet, "/traces/trace-12345", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		// Since mock returned empty rows, expect 404 Not Found
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 for empty mock, got %d", rec.Code)
		}

		// Verify org_id and trace_id were bound to query
		if len(mockConn.lastArgs) < 2 {
			t.Fatalf("expected at least 2 args, got %v", mockConn.lastArgs)
		}
		if mockConn.lastArgs[0] != TestOrgA || mockConn.lastArgs[1] != "trace-12345" {
			t.Errorf("expected args [%s, trace-12345], got %v", TestOrgA, mockConn.lastArgs)
		}
	})
}

func TestSpanIngest_TenantEnrichment(t *testing.T) {
	mockConn := &apiMockConn{}
	traceRepo := chRepo.NewTraceRepository(mockConn)
	handler := NewSpanHandler(traceRepo, nil)

	// Malicious client payload: sets org_id to victim "org-victim-666"
	payload := []domain.Span{
		{
			TraceID:       "trace-1",
			SpanID:        "span-1",
			ServiceName:   "checkout",
			OperationName: "POST /checkout",
			DurationMs:    100,
			StatusCode:    1,
			StartTime:     time.Now(),
			OrgID:         "org-victim-666", // Client attempts spoofing
		},
	}
	body, _ := json.Marshal(payload)

	// Auth context is legitimate user: TestOrgA
	ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
	req := httptest.NewRequest(http.MethodPost, "/spans", bytes.NewReader(body)).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.IngestSpans(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", rec.Code)
	}
}

func TestSpanHandler_IngestOTLP_Protobuf_Gzip_TenantIsolation(t *testing.T) {
	mockConn := &apiMockConn{}
	traceRepo := chRepo.NewTraceRepository(mockConn)
	handler := NewSpanHandler(traceRepo, nil)

	// Malicious client tries to spoof org_id and tenant_id inside OTLP resource and span attributes
	traceIDBytes := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	spanIDBytes := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	protoReq := &coltracepb.ExportTraceServiceRequest{
		ResourceSpans: []*tracepb.ResourceSpans{
			{
				Resource: &resourcepb.Resource{
					Attributes: []*commonpb.KeyValue{
						{
							Key:   "service.name",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "billing-svc"}},
						},
						{
							Key:   "org_id",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "spoofed-victim-org"}},
						},
						{
							Key:   "tenant_id",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "spoofed-victim-tenant"}},
						},
					},
				},
				ScopeSpans: []*tracepb.ScopeSpans{
					{
						Spans: []*tracepb.Span{
							{
								TraceId:           traceIDBytes,
								SpanId:            spanIDBytes,
								Name:              "POST /invoice",
								Kind:              tracepb.Span_SPAN_KIND_SERVER,
								StartTimeUnixNano: uint64(time.Now().Add(-100 * time.Millisecond).UnixNano()),
								EndTimeUnixNano:   uint64(time.Now().UnixNano()),
								Attributes: []*commonpb.KeyValue{
									{
										Key:   "org_id",
										Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "spoofed-victim-org"}},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	protoBytes, err := proto.Marshal(protoReq)
	if err != nil {
		t.Fatalf("failed to marshal proto: %v", err)
	}

	// Compress payload with gzip
	var gzBuffer bytes.Buffer
	gzWriter := gzip.NewWriter(&gzBuffer)
	if _, err := gzWriter.Write(protoBytes); err != nil {
		t.Fatalf("failed to gzip proto: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	// Authentic user is TestOrgA
	ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
	req := httptest.NewRequest(http.MethodPost, "/v1/traces", &gzBuffer).WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept", "application/x-protobuf")

	rec := httptest.NewRecorder()
	handler.IngestOTLP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	if rec.Header().Get("Content-Type") != "application/x-protobuf" {
		t.Errorf("expected Content-Type application/x-protobuf, got %q", rec.Header().Get("Content-Type"))
	}

	// Verify the stored span in ClickHouse batch has TestOrgA as OrgID, not the spoofed org
	if len(mockConn.batches) == 0 || len(mockConn.batches[0].appendedRows) == 0 {
		t.Fatalf("expected batch append to ClickHouse")
	}
	savedOrgID := mockConn.batches[0].appendedRows[0][0].(string)
	if savedOrgID != TestOrgA {
		t.Errorf("SECURITY ALERT: Expected OrgID %q, but got spoofed %q", TestOrgA, savedOrgID)
	}
	savedTags := mockConn.batches[0].appendedRows[0][9].(map[string]string)
	if savedTags["org_id"] != TestOrgA {
		t.Errorf("SECURITY ALERT: Expected tag org_id %q, but got %q", TestOrgA, savedTags["org_id"])
	}
	if _, exists := savedTags["tenant_id"]; exists {
		t.Errorf("SECURITY ALERT: Expected tenant_id tag to be purged, but found %q", savedTags["tenant_id"])
	}
}

func TestAnomaliesHandler_TenantIsolation(t *testing.T) {
	mockConn := &apiMockConn{}
	anomalyRepo := chRepo.NewAnomalyRepository(mockConn)
	handler := NewAnomalyHandler(anomalyRepo)

	t.Run("Rejects ListAnomalies without organization context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/anomalies", nil)
		rec := httptest.NewRecorder()

		handler.ListAnomalies(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Binds context org_id to ListAnomalies", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		req := httptest.NewRequest(http.MethodGet, "/anomalies?limit=25", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.ListAnomalies(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", rec.Code)
		}

		if len(mockConn.lastArgs) == 0 || mockConn.lastArgs[0] != TestOrgA {
			t.Errorf("expected query arg to be %q, got %v", TestOrgA, mockConn.lastArgs)
		}
	})

	t.Run("GetAnomalyByID returns 404 when anomaly does not exist", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		r := chi.NewRouter()
		r.Get("/anomalies/{id}", handler.GetAnomalyByID)

		req := httptest.NewRequest(http.MethodGet, "/anomalies/anm-nonexistent", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", rec.Code)
		}
	})

	t.Run("UpdateAnomalyStatus rejects invalid status values", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		r := chi.NewRouter()
		r.Patch("/anomalies/{id}/status", handler.UpdateAnomalyStatus)

		body := []byte(`{"status": "invalid_status"}`)
		req := httptest.NewRequest(http.MethodPatch, "/anomalies/anm-1/status", bytes.NewReader(body)).WithContext(ctx)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for invalid status, got %d", rec.Code)
		}
	})

	t.Run("UpdateAnomalyStatus accepts valid status values", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		r := chi.NewRouter()
		r.Patch("/anomalies/{id}/status", handler.UpdateAnomalyStatus)

		body := []byte(`{"status": "resolved"}`)
		req := httptest.NewRequest(http.MethodPatch, "/anomalies/anm-1/status", bytes.NewReader(body)).WithContext(ctx)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK for resolved status, got %d", rec.Code)
		}
	})
}

func TestResponseEnvelope(t *testing.T) {
	t.Run("WriteJSON writes correct status and headers", func(t *testing.T) {
		rec := httptest.NewRecorder()
		WriteJSON(rec, http.StatusCreated, map[string]string{"foo": "bar"})

		if rec.Code != http.StatusCreated {
			t.Errorf("expected 201 Created, got %d", rec.Code)
		}
		if rec.Header().Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", rec.Header().Get("Content-Type"))
		}
	})

	t.Run("WriteError serializes error message envelope", func(t *testing.T) {
		rec := httptest.NewRecorder()
		WriteError(rec, http.StatusBadRequest, "sample error")

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rec.Code)
		}
		var body map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if body["error"] != "sample error" {
			t.Errorf("expected error message in envelope, got %v", body)
		}
	})
}

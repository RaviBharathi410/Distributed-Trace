package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/auth"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	chRepo "github.com/RaviBharathi410/distributedtrace/internal/repository/clickhouse"
	"github.com/go-chi/chi/v5"
)

func TestAuthHandler_ValidationAndEdgeCases(t *testing.T) {
	// AuthHandler with nil repos for testing pure validation and parsing logic
	handler := NewAuthHandler(nil, nil, nil, nil)

	t.Run("Register rejects malformed json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte("{invalid-json")))
		rec := httptest.NewRecorder()

		handler.Register(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	t.Run("Register rejects empty email or password", func(t *testing.T) {
		cases := []string{
			`{"email":"","password":"ValidPassword123!"}`,
			`{"email":"user@example.com","password":""}`,
			`{"email":"","password":""}`,
		}

		for _, payload := range cases {
			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte(payload)))
			rec := httptest.NewRecorder()

			handler.Register(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for payload %s, got %d", payload, rec.Code)
			}
		}
	})

	t.Run("Login rejects malformed json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte("{invalid")))
		rec := httptest.NewRecorder()

		handler.Login(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	t.Run("Me rejects invalid session user id", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.UserIDKey, "not-a-uuid")
		req := httptest.NewRequest(http.MethodGet, "/auth/me", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.Me(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for malformed user uuid, got %d", rec.Code)
		}
	})

	t.Run("CreateAPIKey rejects missing or invalid organization id", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, "invalid-org-uuid")
		req := httptest.NewRequest(http.MethodPost, "/auth/api-keys", bytes.NewReader([]byte(`{"name":"test"}`))).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.CreateAPIKey(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for invalid org uuid, got %d", rec.Code)
		}
	})

	t.Run("ListAPIKeys rejects invalid organization id", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, "not-a-valid-uuid")
		req := httptest.NewRequest(http.MethodGet, "/auth/api-keys", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.ListAPIKeys(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for invalid org uuid, got %d", rec.Code)
		}
	})

	t.Run("RevokeAPIKey rejects invalid key id parameter", func(t *testing.T) {
		r := chi.NewRouter()
		r.Delete("/auth/api-keys/{id}", handler.RevokeAPIKey)

		req := httptest.NewRequest(http.MethodDelete, "/auth/api-keys/not-a-uuid", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for non-UUID key ID, got %d", rec.Code)
		}
	})
}

func TestServicesHandler_StatsAndValidation(t *testing.T) {
	mockConn := &apiMockConn{}
	serviceRepo := chRepo.NewServiceRepository(mockConn)
	handler := NewServiceHandler(serviceRepo)

	t.Run("GetServiceStats rejects missing org context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/services/checkout/stats", nil)
		rec := httptest.NewRecorder()

		handler.GetServiceStats(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("GetServiceStats rejects empty service name parameter", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		r := chi.NewRouter()
		r.Get("/services/{service}/stats", handler.GetServiceStats)

		// Request without matching param
		req := httptest.NewRequest(http.MethodGet, "/services//stats", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.GetServiceStats(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request when service param is empty, got %d", rec.Code)
		}
	})

	t.Run("GetServiceStats succeeds with valid service parameter and time queries", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		r := chi.NewRouter()
		r.Get("/services/{service}/stats", handler.GetServiceStats)

		req := httptest.NewRequest(http.MethodGet, "/services/checkout-svc/stats?from=2026-09-27T00:00:00Z&to=2026-09-27T01:00:00Z", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK for valid stats request, got %d", rec.Code)
		}
	})

	t.Run("GetServiceGraph rejects missing org context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/services/graph", nil)
		rec := httptest.NewRecorder()

		handler.GetServiceGraph(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("GetServiceGraph supports custom RFC3339 from/to parameters", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		req := httptest.NewRequest(http.MethodGet, "/services/graph?from=2026-09-27T10:00:00Z&to=2026-09-27T12:00:00Z", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.GetServiceGraph(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", rec.Code)
		}
	})
}

func TestTracesHandler_EdgeCases(t *testing.T) {
	mockConn := &apiMockConn{}
	traceRepo := chRepo.NewTraceRepository(mockConn)
	handler := NewTraceHandler(traceRepo)

	t.Run("GetTraceByID rejects missing org context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/traces/trace-123", nil)
		rec := httptest.NewRecorder()

		handler.GetTraceByID(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("GetTraceByID rejects missing trace id param", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		req := httptest.NewRequest(http.MethodGet, "/traces/", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.GetTraceByID(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	t.Run("SearchTraces parses query parameters correctly", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		req := httptest.NewRequest(http.MethodGet, "/traces?service=checkout&min_duration_ms=100&max_duration_ms=500&limit=25&offset=10&from=2026-09-27T00:00:00Z&to=2026-09-27T01:00:00Z", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.SearchTraces(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", rec.Code)
		}
	})
}

func TestSpanIngest_ValidationBranches(t *testing.T) {
	mockConn := &apiMockConn{}
	traceRepo := chRepo.NewTraceRepository(mockConn)
	handler := NewSpanHandler(traceRepo, nil)

	t.Run("Rejects ingest when org context is missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/spans", bytes.NewReader([]byte(`[]`)))
		rec := httptest.NewRecorder()

		handler.IngestSpans(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Rejects ingest with malformed JSON", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		req := httptest.NewRequest(http.MethodPost, "/spans", bytes.NewReader([]byte(`{not-json-array`))).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.IngestSpans(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	t.Run("Rejects ingest with empty span list", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		req := httptest.NewRequest(http.MethodPost, "/spans", bytes.NewReader([]byte(`[]`))).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.IngestSpans(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	t.Run("Rejects ingest with invalid span schema", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		// Missing TraceID and ServiceName
		spans := []domain.Span{
			{
				SpanID:    "span-1",
				StartTime: time.Now(),
			},
		}
		body, _ := json.Marshal(spans)
		req := httptest.NewRequest(http.MethodPost, "/spans", bytes.NewReader(body)).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.IngestSpans(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request on invalid telemetry schema, got %d", rec.Code)
		}
	})
}

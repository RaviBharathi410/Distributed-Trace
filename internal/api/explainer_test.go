package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/RaviBharathi410/distributedtrace/internal/auth"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	chRepo "github.com/RaviBharathi410/distributedtrace/internal/repository/clickhouse"
	"github.com/go-chi/chi/v5"
)

type mockAnomalyExplainer struct {
	called       bool
	forceRefresh bool
	res          *domain.IncidentExplanation
	err          error
}

func (m *mockAnomalyExplainer) Explain(ctx context.Context, orgID string, anomaly *domain.Anomaly, forceRefresh bool) (*domain.IncidentExplanation, error) {
	m.called = true
	m.forceRefresh = forceRefresh
	if m.err != nil {
		return nil, m.err
	}
	return m.res, nil
}

type explainerMockConn struct {
	hasRow   bool
	anomaly  domain.Anomaly
	lastArgs []any
}

func (m *explainerMockConn) Contributors() []string                         { return nil }
func (m *explainerMockConn) ServerVersion() (*driver.ServerVersion, error) { return nil, nil }
func (m *explainerMockConn) Select(ctx context.Context, dest any, query string, args ...any) error {
	return nil
}
func (m *explainerMockConn) Ping(ctx context.Context) error { return nil }
func (m *explainerMockConn) Stats() driver.Stats            { return driver.Stats{} }
func (m *explainerMockConn) Close() error                   { return nil }
func (m *explainerMockConn) AsyncInsert(ctx context.Context, query string, wait bool, args ...any) error {
	return nil
}
func (m *explainerMockConn) PrepareBatch(ctx context.Context, query string, opts ...driver.PrepareBatchOption) (driver.Batch, error) {
	return nil, nil
}
func (m *explainerMockConn) Exec(ctx context.Context, query string, args ...any) error {
	return nil
}
func (m *explainerMockConn) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	return nil
}
func (m *explainerMockConn) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	m.lastArgs = args
	return &explainerMockRows{
		hasRow:  m.hasRow,
		anomaly: m.anomaly,
	}, nil
}

type explainerMockRows struct {
	hasRow  bool
	read    bool
	anomaly domain.Anomaly
}

func (r *explainerMockRows) Next() bool {
	if r.hasRow && !r.read {
		r.read = true
		return true
	}
	return false
}

func (r *explainerMockRows) Scan(dest ...any) error {
	*(dest[0].(*string)) = r.anomaly.ID
	*(dest[1].(*string)) = r.anomaly.OrgID
	*(dest[2].(*string)) = r.anomaly.ServiceName
	*(dest[3].(*string)) = r.anomaly.OperationName
	*(dest[4].(*string)) = r.anomaly.RootCauseService
	*(dest[5].(*string)) = string(r.anomaly.Severity)
	*(dest[6].(*float64)) = r.anomaly.ZScore
	*(dest[7].(*float64)) = r.anomaly.BaselineLatencyMs
	*(dest[8].(*float64)) = r.anomaly.ObservedLatencyMs
	*(dest[9].(*float64)) = r.anomaly.ErrorRateDelta
	*(dest[10].(*string)) = r.anomaly.Status
	*(dest[11].(*time.Time)) = r.anomaly.DetectedAt
	*(dest[12].(**time.Time)) = r.anomaly.ResolvedAt
	*(dest[13].(*[]string)) = r.anomaly.RootCausePath
	return nil
}

func (r *explainerMockRows) ScanStruct(dest any) error       { return nil }
func (r *explainerMockRows) ColumnTypes() []driver.ColumnType { return nil }
func (r *explainerMockRows) Totals(dest ...any) error        { return nil }
func (r *explainerMockRows) Columns() []string               { return nil }
func (r *explainerMockRows) Close() error                    { return nil }
func (r *explainerMockRows) Err() error                      { return nil }

func TestAnomalyHandler_ExplainAnomaly(t *testing.T) {
	sampleAnm := domain.Anomaly{
		ID:                "anm-1",
		OrgID:             TestOrgA,
		ServiceName:       "checkout-api",
		OperationName:     "POST /order",
		RootCauseService:  "payment-svc",
		Severity:          domain.SeverityCritical,
		ZScore:            4.5,
		BaselineLatencyMs: 20.0,
		ObservedLatencyMs: 150.0,
		ErrorRateDelta:    0.5,
		Status:            "open",
		DetectedAt:        time.Now(),
		RootCausePath:     []string{"checkout-api", "payment-svc"},
	}

	t.Run("Rejects request without organization context (401)", func(t *testing.T) {
		mockConn := &explainerMockConn{}
		anomalyRepo := chRepo.NewAnomalyRepository(mockConn)
		handler := NewAnomalyHandler(anomalyRepo)

		req := httptest.NewRequest(http.MethodPost, "/anomalies/anm-1/explain", nil)
		rec := httptest.NewRecorder()

		handler.ExplainAnomaly(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Returns 404 when anomaly does not exist for tenant", func(t *testing.T) {
		mockConn := &explainerMockConn{hasRow: false}
		anomalyRepo := chRepo.NewAnomalyRepository(mockConn)
		handler := NewAnomalyHandler(anomalyRepo)

		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		r := chi.NewRouter()
		r.Post("/anomalies/{id}/explain", handler.ExplainAnomaly)

		req := httptest.NewRequest(http.MethodPost, "/anomalies/anm-nonexistent/explain", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", rec.Code)
		}
	})

	t.Run("Returns 503 when explainer engine is nil", func(t *testing.T) {
		mockConn := &explainerMockConn{hasRow: true, anomaly: sampleAnm}
		anomalyRepo := chRepo.NewAnomalyRepository(mockConn)
		handler := NewAnomalyHandler(anomalyRepo) // nil explainer

		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		r := chi.NewRouter()
		r.Post("/anomalies/{id}/explain", handler.ExplainAnomaly)

		req := httptest.NewRequest(http.MethodPost, "/anomalies/anm-1/explain", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503 Service Unavailable, got %d", rec.Code)
		}
	})

	t.Run("Successfully explains anomaly and returns 200 OK", func(t *testing.T) {
		mockConn := &explainerMockConn{hasRow: true, anomaly: sampleAnm}
		anomalyRepo := chRepo.NewAnomalyRepository(mockConn)

		expectedExp := &domain.IncidentExplanation{
			AnomalyID:               "anm-1",
			OrgID:                   TestOrgA,
			RootCauseService:        "payment-svc",
			OperationName:           "POST /order",
			ConfidenceScore:         0.94,
			Summary:                 "Database lock wait in payment-svc spiked latency.",
			ContributingFactors:     []string{"Factor 1", "Factor 2"},
			DegradedToDeterministic: false,
			CostAttribution: domain.ExplanationCost{
				Model:            "gemini-1.5-flash",
				InputTokens:      820,
				OutputTokens:     160,
				EstimatedCostUSD: 0.000109,
				CostCeilingUSD:   0.010,
				HourlyCapUSD:     1.000,
			},
			GeneratedAt: time.Now(),
		}

		mockExplainer := &mockAnomalyExplainer{res: expectedExp}
		handler := NewAnomalyHandlerWithExplainer(anomalyRepo, mockExplainer)

		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		r := chi.NewRouter()
		r.Post("/anomalies/{id}/explain", handler.ExplainAnomaly)

		body := strings.NewReader(`{"force_refresh": true}`)
		req := httptest.NewRequest(http.MethodPost, "/anomalies/anm-1/explain", body).WithContext(ctx)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		if !mockExplainer.called {
			t.Errorf("expected mock explainer to be called")
		}
		if !mockExplainer.forceRefresh {
			t.Errorf("expected forceRefresh to be true from request body")
		}

		var resp domain.IncidentExplanation
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response json: %v", err)
		}

		if resp.AnomalyID != "anm-1" {
			t.Errorf("expected anomaly_id 'anm-1', got %s", resp.AnomalyID)
		}
		if resp.Summary != expectedExp.Summary {
			t.Errorf("expected summary %q, got %q", expectedExp.Summary, resp.Summary)
		}
		if resp.CostAttribution.Model != "gemini-1.5-flash" {
			t.Errorf("expected model 'gemini-1.5-flash', got %s", resp.CostAttribution.Model)
		}
	})
}

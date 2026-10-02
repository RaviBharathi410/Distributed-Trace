package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/auth"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	pgRepo "github.com/RaviBharathi410/distributedtrace/internal/repository/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type apiMockPgxRow struct {
	values []any
}

func (r *apiMockPgxRow) Scan(dest ...any) error {
	for i, val := range r.values {
		if i < len(dest) {
			switch d := dest[i].(type) {
			case *float64:
				if v, ok := val.(float64); ok {
					*d = v
				}
			case *int:
				if v, ok := val.(int); ok {
					*d = v
				}
			}
		}
	}
	return nil
}

type apiMockPgxRows struct {
	rows    [][]any
	currIdx int
}

func (r *apiMockPgxRows) Close() {}
func (r *apiMockPgxRows) Err() error { return nil }
func (r *apiMockPgxRows) CommandTag() pgconn.CommandTag { return pgconn.NewCommandTag("SELECT 1") }
func (r *apiMockPgxRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *apiMockPgxRows) Next() bool {
	if r.currIdx < len(r.rows) {
		r.currIdx++
		return true
	}
	return false
}
func (r *apiMockPgxRows) Scan(dest ...any) error {
	row := r.rows[r.currIdx-1]
	for i, val := range row {
		if i < len(dest) {
			switch d := dest[i].(type) {
			case *string:
				if v, ok := val.(string); ok {
					*d = v
				}
			case *int:
				if v, ok := val.(int); ok {
					*d = v
				}
			case *float64:
				if v, ok := val.(float64); ok {
					*d = v
				}
			case *time.Time:
				if v, ok := val.(time.Time); ok {
					*d = v
				}
			}
		}
	}
	return nil
}
func (r *apiMockPgxRows) Values() ([]any, error) { return nil, nil }
func (r *apiMockPgxRows) RawValues() [][]byte    { return nil }
func (r *apiMockPgxRows) Conn() *pgx.Conn        { return nil }

type apiMockPgxPool struct {
	queriedOrg string
}

func (m *apiMockPgxPool) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (m *apiMockPgxPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if len(args) > 0 {
		if s, ok := args[0].(string); ok {
			m.queriedOrg = s
		}
	}

	if strings.Contains(sql, "COALESCE(SUM(estimated_cost_usd), 0.0)") && strings.Contains(sql, "COUNT(*)") {
		if m.queriedOrg == "org-alpha" {
			return &apiMockPgxRow{values: []any{0.0050, 15, 0.00033}}
		}
		return &apiMockPgxRow{values: []any{0.0, 0, 0.0}}
	}

	if strings.Contains(sql, "created_at >=") {
		if m.queriedOrg == "org-alpha" {
			return &apiMockPgxRow{values: []any{0.0020}}
		}
		return &apiMockPgxRow{values: []any{0.0}}
	}

	if strings.Contains(sql, "FROM organizations") {
		return &apiMockPgxRow{values: []any{3}}
	}

	return &apiMockPgxRow{values: []any{}}
}

func (m *apiMockPgxPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if len(args) > 0 {
		if s, ok := args[0].(string); ok {
			m.queriedOrg = s
		}
	}

	if strings.Contains(sql, "FROM tenant_llm_costs") && m.queriedOrg == "org-alpha" {
		return &apiMockPgxRows{
			rows: [][]any{
				{"c1", "anm-1", "gemini-1.5-flash", 900, 200, 0.000125, time.Now()},
			},
		}, nil
	}

	return &apiMockPgxRows{rows: [][]any{}}, nil
}

func TestCostHandler_Summary_TenantIsolation(t *testing.T) {
	pool := &apiMockPgxPool{}
	repo := pgRepo.NewCostRepository(pool)
	handler := NewCostHandler(repo)

	// 1. Missing Auth -> 401
	reqUnauth := httptest.NewRequest("GET", "/api/v1/costs/summary", nil)
	rrUnauth := httptest.NewRecorder()
	handler.GetCostSummary(rrUnauth, reqUnauth)
	if rrUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated request, got %d", rrUnauth.Code)
	}

	// 2. Org Alpha Request -> Returns Org Alpha Summary
	reqA := httptest.NewRequest("GET", "/api/v1/costs/summary", nil)
	ctxA := context.WithValue(reqA.Context(), auth.OrgIDKey, "org-alpha")
	reqA = reqA.WithContext(ctxA)
	rrA := httptest.NewRecorder()

	handler.GetCostSummary(rrA, reqA)
	if rrA.Code != http.StatusOK {
		t.Fatalf("expected 200 for Org Alpha, got %d: %s", rrA.Code, rrA.Body.String())
	}

	var summaryA domain.CostSummary
	if err := json.Unmarshal(rrA.Body.Bytes(), &summaryA); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if summaryA.OrgID != "org-alpha" {
		t.Errorf("expected orgId 'org-alpha', got %s", summaryA.OrgID)
	}
	if summaryA.LLMSpendUSD != 0.0050 {
		t.Errorf("expected LLM spend 0.0050, got %f", summaryA.LLMSpendUSD)
	}
	if summaryA.TotalIncidentsExplained != 15 {
		t.Errorf("expected 15 incidents, got %d", summaryA.TotalIncidentsExplained)
	}
	if summaryA.InfraShareUSD != 5.00 { // 15.00 / 3 active orgs
		t.Errorf("expected infra share 5.00, got %f", summaryA.InfraShareUSD)
	}

	// 3. Org Beta Request -> Returns Org Beta Summary (Zero spend)
	reqB := httptest.NewRequest("GET", "/api/v1/costs/summary", nil)
	ctxB := context.WithValue(reqB.Context(), auth.OrgIDKey, "org-beta")
	reqB = reqB.WithContext(ctxB)
	rrB := httptest.NewRecorder()

	handler.GetCostSummary(rrB, reqB)
	if rrB.Code != http.StatusOK {
		t.Fatalf("expected 200 for Org Beta, got %d", rrB.Code)
	}

	var summaryB domain.CostSummary
	if err := json.Unmarshal(rrB.Body.Bytes(), &summaryB); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if summaryB.OrgID != "org-beta" {
		t.Errorf("expected orgId 'org-beta', got %s", summaryB.OrgID)
	}
	if summaryB.LLMSpendUSD != 0.0 {
		t.Errorf("CROSS-TENANT LEAK: Org Beta saw LLM spend: %f", summaryB.LLMSpendUSD)
	}
	if summaryB.TotalIncidentsExplained != 0 {
		t.Errorf("CROSS-TENANT LEAK: Org Beta saw incidents: %d", summaryB.TotalIncidentsExplained)
	}
}

func TestCostHandler_Breakdown_TenantIsolation(t *testing.T) {
	pool := &apiMockPgxPool{}
	repo := pgRepo.NewCostRepository(pool)
	handler := NewCostHandler(repo)

	// Org Alpha Request -> Returns 1 item
	reqA := httptest.NewRequest("GET", "/api/v1/costs/breakdown", nil)
	ctxA := context.WithValue(reqA.Context(), auth.OrgIDKey, "org-alpha")
	reqA = reqA.WithContext(ctxA)
	rrA := httptest.NewRecorder()

	handler.GetCostBreakdown(rrA, reqA)
	if rrA.Code != http.StatusOK {
		t.Fatalf("expected 200 for Org Alpha breakdown, got %d", rrA.Code)
	}

	var respA struct {
		Items []domain.CostBreakdownItem `json:"items"`
	}
	if err := json.Unmarshal(rrA.Body.Bytes(), &respA); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(respA.Items) != 1 {
		t.Errorf("expected 1 item for Org Alpha, got %d", len(respA.Items))
	}

	// Org Beta Request -> Returns 0 items
	reqB := httptest.NewRequest("GET", "/api/v1/costs/breakdown", nil)
	ctxB := context.WithValue(reqB.Context(), auth.OrgIDKey, "org-beta")
	reqB = reqB.WithContext(ctxB)
	rrB := httptest.NewRecorder()

	handler.GetCostBreakdown(rrB, reqB)
	if rrB.Code != http.StatusOK {
		t.Fatalf("expected 200 for Org Beta breakdown, got %d", rrB.Code)
	}

	var respB struct {
		Items []domain.CostBreakdownItem `json:"items"`
	}
	if err := json.Unmarshal(rrB.Body.Bytes(), &respB); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(respB.Items) != 0 {
		t.Errorf("CROSS-TENANT LEAK: Org Beta breakdown returned items: %d", len(respB.Items))
	}
}

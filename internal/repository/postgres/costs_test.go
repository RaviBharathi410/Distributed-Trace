package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type mockPgxRow struct {
	values []any
	err    error
}

func (r *mockPgxRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
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
			case *string:
				if v, ok := val.(string); ok {
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

type mockPgxRows struct {
	rows    [][]any
	currIdx int
	closed  bool
}

func (r *mockPgxRows) Close() {
	r.closed = true
}

func (r *mockPgxRows) Err() error {
	return nil
}

func (r *mockPgxRows) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT 1")
}

func (r *mockPgxRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *mockPgxRows) Next() bool {
	if r.currIdx < len(r.rows) {
		r.currIdx++
		return true
	}
	return false
}

func (r *mockPgxRows) Scan(dest ...any) error {
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

func (r *mockPgxRows) Values() ([]any, error) {
	return nil, nil
}

func (r *mockPgxRows) RawValues() [][]byte {
	return nil
}

func (r *mockPgxRows) Conn() *pgx.Conn {
	return nil
}

type mockPgxPool struct {
	lastSQL  string
	lastArgs []any
	allSQL   []string
}

func (m *mockPgxPool) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	m.lastSQL = sql
	m.lastArgs = arguments
	m.allSQL = append(m.allSQL, sql)
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (m *mockPgxPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	m.lastSQL = sql
	m.lastArgs = args
	m.allSQL = append(m.allSQL, sql)

	orgID := ""
	if len(args) > 0 {
		if s, ok := args[0].(string); ok {
			orgID = s
		}
	}

	// 1. Spend summary query
	if strings.Contains(sql, "COALESCE(SUM(estimated_cost_usd), 0.0)") && strings.Contains(sql, "COUNT(*)") {
		if orgID == "org-a" {
			return &mockPgxRow{values: []any{0.0025, 10, 0.00025}}
		} else if orgID == "org-b" {
			return &mockPgxRow{values: []any{0.0, 0, 0.0}}
		}
		return &mockPgxRow{values: []any{0.0, 0, 0.0}}
	}

	// 2. Hourly spend query
	if strings.Contains(sql, "COALESCE(SUM(estimated_cost_usd), 0.0)") && strings.Contains(sql, "created_at >=") {
		if orgID == "org-a" {
			return &mockPgxRow{values: []any{0.0010}}
		}
		return &mockPgxRow{values: []any{0.0}}
	}

	// 3. Active orgs count
	if strings.Contains(sql, "COUNT(*)") && strings.Contains(sql, "FROM organizations") {
		return &mockPgxRow{values: []any{5}}
	}

	return &mockPgxRow{values: []any{}}
}

func (m *mockPgxPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	m.lastSQL = sql
	m.lastArgs = args

	orgID := ""
	if len(args) > 0 {
		if s, ok := args[0].(string); ok {
			orgID = s
		}
	}

	if strings.Contains(sql, "FROM tenant_llm_costs") && orgID == "org-a" {
		return &mockPgxRows{
			rows: [][]any{
				{"cost-01", "anm-01", "gemini-1.5-flash", 800, 150, 0.000105, time.Now()},
				{"cost-02", "anm-02", "gemini-1.5-flash", 920, 200, 0.000129, time.Now()},
			},
		}, nil
	}

	return &mockPgxRows{rows: [][]any{}}, nil
}

func TestCostRepository_TenantIsolation(t *testing.T) {
	mockPool := &mockPgxPool{}
	repo := NewCostRepository(mockPool)
	ctx := context.Background()

	// 1. Org A Summary
	summaryA, err := repo.GetCostSummary(ctx, "org-a")
	if err != nil {
		t.Fatalf("GetCostSummary failed for org-a: %v", err)
	}

	foundSpendQuery := false
	for _, q := range mockPool.allSQL {
		if strings.Contains(q, "FROM tenant_llm_costs") {
			if !strings.Contains(q, "org_id = $1") {
				t.Fatalf("CRITICAL SECURITY LEAK: tenant_llm_costs query missing org_id = $1: %s", q)
			}
			foundSpendQuery = true
		}
	}
	if !foundSpendQuery {
		t.Fatal("expected at least one query against tenant_llm_costs")
	}
	if summaryA.OrgID != "org-a" {
		t.Errorf("expected OrgID 'org-a', got %s", summaryA.OrgID)
	}
	if summaryA.LLMSpendUSD != 0.0025 {
		t.Errorf("expected LLM spend 0.0025, got %f", summaryA.LLMSpendUSD)
	}
	if summaryA.TotalIncidentsExplained != 10 {
		t.Errorf("expected 10 incidents, got %d", summaryA.TotalIncidentsExplained)
	}
	if summaryA.InfraShareUSD != 3.00 { // 15.00 / 5 active orgs
		t.Errorf("expected 3.00 infra share for 5 active orgs, got %f", summaryA.InfraShareUSD)
	}
	if summaryA.TotalCostUSD != 3.0025 {
		t.Errorf("expected total cost 3.0025, got %f", summaryA.TotalCostUSD)
	}

	// 2. Org B Summary (Assert strict isolation - zero spend)
	summaryB, err := repo.GetCostSummary(ctx, "org-b")
	if err != nil {
		t.Fatalf("GetCostSummary failed for org-b: %v", err)
	}
	if summaryB.LLMSpendUSD != 0.0 {
		t.Errorf("CROSS-TENANT LEAK: Org B saw LLM spend: %f", summaryB.LLMSpendUSD)
	}
	if summaryB.TotalIncidentsExplained != 0 {
		t.Errorf("CROSS-TENANT LEAK: Org B saw explained incidents: %d", summaryB.TotalIncidentsExplained)
	}

	// 3. Breakdown isolation
	itemsA, err := repo.GetCostBreakdown(ctx, "org-a", 10, 0)
	if err != nil {
		t.Fatalf("GetCostBreakdown failed for org-a: %v", err)
	}
	if len(itemsA) != 2 {
		t.Errorf("expected 2 items for org-a, got %d", len(itemsA))
	}

	itemsB, err := repo.GetCostBreakdown(ctx, "org-b", 10, 0)
	if err != nil {
		t.Fatalf("GetCostBreakdown failed for org-b: %v", err)
	}
	if len(itemsB) != 0 {
		t.Errorf("CROSS-TENANT LEAK: Org B saw cost items: %d", len(itemsB))
	}
}

func TestCostRepository_RecordCost(t *testing.T) {
	mockPool := &mockPgxPool{}
	repo := NewCostRepository(mockPool)
	ctx := context.Background()

	event := &domain.LLMCostEvent{
		OrgID:            "org-test-01",
		AnomalyID:        "anm-test-01",
		Model:            "gemini-1.5-flash",
		InputTokens:      1000,
		OutputTokens:     250,
		EstimatedCostUSD: 0.000150,
	}

	if err := repo.RecordCost(ctx, event); err != nil {
		t.Fatalf("RecordCost failed: %v", err)
	}

	if !strings.Contains(mockPool.lastSQL, "INSERT INTO tenant_llm_costs") {
		t.Errorf("expected INSERT query, got: %s", mockPool.lastSQL)
	}
	if mockPool.lastArgs[1] != "org-test-01" {
		t.Errorf("expected org-test-01 arg, got: %v", mockPool.lastArgs[1])
	}
}

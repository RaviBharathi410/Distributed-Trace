package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
)

const (
	OrgA = "org-alpha-111"
	OrgB = "org-beta-222"
)

// --- Mock ClickHouse Driver Structures ---

type mockRow struct {
	values []any
	err    error
}

func (r *mockRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(r.values) == 0 {
		return fmt.Errorf("no row found")
	}
	for i, val := range r.values {
		if i >= len(dest) {
			break
		}
		assignVal(dest[i], val)
	}
	return nil
}

func (r *mockRow) ScanStruct(dest any) error { return nil }
func (r *mockRow) Err() error                { return r.err }

type mockRows struct {
	rows [][]any
	idx  int
}

func (r *mockRows) Next() bool {
	if r.idx < len(r.rows) {
		r.idx++
		return true
	}
	return false
}

func (r *mockRows) Scan(dest ...any) error {
	curr := r.rows[r.idx-1]
	for i, val := range curr {
		if i >= len(dest) {
			break
		}
		assignVal(dest[i], val)
	}
	return nil
}

func assignVal(target any, source any) {
	switch t := target.(type) {
	case *string:
		if s, ok := source.(string); ok {
			*t = s
		}
	case *uint8:
		if u, ok := source.(uint8); ok {
			*t = u
		}
	case *uint32:
		if u, ok := source.(uint32); ok {
			*t = u
		}
	case *uint64:
		if u, ok := source.(uint64); ok {
			*t = u
		} else if n, ok := source.(int); ok {
			*t = uint64(n)
		}
	case *map[string]string:
		if m, ok := source.(map[string]string); ok {
			*t = m
		}
	case *int:
		if n, ok := source.(int); ok {
			*t = n
		}
	case *int64:
		if n, ok := source.(int64); ok {
			*t = n
		}
	case *float64:
		if f, ok := source.(float64); ok {
			*t = f
		}
	case *time.Time:
		if tm, ok := source.(time.Time); ok {
			*t = tm
		}
	case **time.Time:
		if tm, ok := source.(*time.Time); ok {
			*t = tm
		}
	case *[]string:
		if arr, ok := source.([]string); ok {
			*t = arr
		}
	}
}

func (r *mockRows) ScanStruct(dest any) error            { return nil }
func (r *mockRows) ColumnTypes() []driver.ColumnType      { return nil }
func (r *mockRows) Totals(dest ...any) error              { return nil }
func (r *mockRows) Columns() []string                     { return nil }
func (r *mockRows) Close() error                          { return nil }
func (r *mockRows) Err() error                            { return nil }

type mockBatch struct {
	appendedRows [][]any
}

func (b *mockBatch) Abort() error               { return nil }
func (b *mockBatch) Append(v ...any) error       { b.appendedRows = append(b.appendedRows, v); return nil }
func (b *mockBatch) AppendStruct(v any) error    { return nil }
func (b *mockBatch) Column(int) driver.BatchColumn { return nil }
func (b *mockBatch) Flush() error                { return nil }
func (b *mockBatch) Send() error                 { return nil }
func (b *mockBatch) IsSent() bool                { return true }
func (b *mockBatch) Rows() int                   { return len(b.appendedRows) }

type MockConn struct {
	LastQuery string
	LastArgs  []any
}

func (m *MockConn) Contributors() []string                          { return nil }
func (m *MockConn) ServerVersion() (*driver.ServerVersion, error)  { return nil, nil }
func (m *MockConn) Select(ctx context.Context, dest any, query string, args ...any) error {
	return nil
}
func (m *MockConn) Ping(ctx context.Context) error                 { return nil }
func (m *MockConn) Stats() driver.Stats                             { return driver.Stats{} }
func (m *MockConn) Close() error                                    { return nil }
func (m *MockConn) AsyncInsert(ctx context.Context, query string, wait bool, args ...any) error {
	return nil
}

func (m *MockConn) PrepareBatch(ctx context.Context, query string, opts ...driver.PrepareBatchOption) (driver.Batch, error) {
	return &mockBatch{}, nil
}

func (m *MockConn) Exec(ctx context.Context, query string, args ...any) error {
	m.LastQuery = query
	m.LastArgs = args
	return nil
}

func (m *MockConn) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	m.LastQuery = query
	m.LastArgs = args

	// Route service stats query
	if strings.Contains(query, "quantile(0.50)(duration_ms)") {
		return &mockRow{
			values: []any{45.0, 90.0, 150.0, 0.5, 120.0},
		}
	}

	// Route anomaly stats query (ClickHouse sum returns UInt64)
	if strings.Contains(query, "sum(severity = 'critical')") {
		return &mockRow{
			values: []any{uint64(1), uint64(2), uint64(3), uint64(4)},
		}
	}

	return &mockRow{}
}

func (m *MockConn) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	m.LastQuery = query
	m.LastArgs = args

	// Extract org_id from query args
	var queriedOrg string
	for _, arg := range args {
		if str, ok := arg.(string); ok && (str == OrgA || str == OrgB) {
			queriedOrg = str
			break
		}
	}

	// 1. ListServices
	if strings.Contains(query, "SELECT DISTINCT service_name FROM otel_spans") {
		if queriedOrg == OrgA {
			return &mockRows{
				rows: [][]any{
					{"checkout-api"},
					{"inventory-svc"},
				},
			}, nil
		} else if queriedOrg == OrgB {
			return &mockRows{
				rows: [][]any{
					{"secret-billing-svc"},
					{"payroll-svc"},
				},
			}, nil
		}
		return &mockRows{rows: [][]any{}}, nil
	}

	// 2. GetServiceGraph Edges
	if strings.Contains(query, "GROUP BY source, target") {
		if queriedOrg == OrgA {
			return &mockRows{
				rows: [][]any{
					{"checkout-api", "inventory-svc", 45.0, 0.1},
				},
			}, nil
		} else if queriedOrg == OrgB {
			return &mockRows{
				rows: [][]any{
					{"secret-billing-svc", "payroll-svc", 10.0, 0.0},
				},
			}, nil
		}
		return &mockRows{rows: [][]any{}}, nil
	}

	// 3. GetServiceGraph Nodes
	if strings.Contains(query, "GROUP BY service_name") {
		if queriedOrg == OrgA {
			return &mockRows{
				rows: [][]any{
					{"checkout-api", 50.0, 0.1},
					{"inventory-svc", 35.0, 0.0},
				},
			}, nil
		} else if queriedOrg == OrgB {
			return &mockRows{
				rows: [][]any{
					{"secret-billing-svc", 200.0, 0.0},
					{"payroll-svc", 120.0, 0.0},
				},
			}, nil
		}
		return &mockRows{rows: [][]any{}}, nil
	}

	// 4. SearchTraces
	if strings.Contains(query, "SELECT trace_id,") && strings.Contains(query, "GROUP BY trace_id") {
		if queriedOrg == OrgA {
			return &mockRows{
				rows: [][]any{
					{"trace-alpha-001", "checkout-api", "process_checkout", uint32(120), 4, 0, int64(1700000000000)},
				},
			}, nil
		} else if queriedOrg == OrgB {
			return &mockRows{
				rows: [][]any{
					{"trace-beta-999", "secret-billing-svc", "charge", uint32(500), 2, 1, int64(1700000000000)},
				},
			}, nil
		}
		return &mockRows{rows: [][]any{}}, nil
	}

	// 5. GetTraceWithSpans
	if strings.Contains(query, "WHERE org_id = ? AND trace_id = ?") {
		orgIDArg := args[0].(string)
		traceIDArg := args[1].(string)

		if orgIDArg == OrgA && traceIDArg == "trace-alpha-001" {
			return &mockRows{
				rows: [][]any{
					{"trace-alpha-001", "span-1", "", "checkout-api", "process_checkout", uint32(120), uint8(1), "", map[string]string{"env": "test"}, time.Now()},
				},
			}, nil
		}
		// If Org A queries Org B's trace or vice versa, return empty
		return &mockRows{rows: [][]any{}}, nil
	}

	// 6. Anomalies List
	if strings.Contains(query, "FROM anomalies") {
		if queriedOrg == OrgA {
			return &mockRows{
				rows: [][]any{
					{"anm-alpha-01", OrgA, "checkout-api", "process_checkout", "checkout-api", "critical", 3.5, 50.0, 200.0, 15.0, "open", time.Now(), nil, []string{"checkout-api"}},
				},
			}, nil
		} else if queriedOrg == OrgB {
			return &mockRows{
				rows: [][]any{
					{"anm-beta-99", OrgB, "secret-billing-svc", "charge", "secret-billing-svc", "high", 2.8, 100.0, 350.0, 5.0, "open", time.Now(), nil, []string{"secret-billing-svc"}},
				},
			}, nil
		}
		return &mockRows{rows: [][]any{}}, nil
	}

	return &mockRows{rows: [][]any{}}, nil
}

// --- Multi-Tenant Isolation Tests ---

func TestListServices_TenantIsolation(t *testing.T) {
	conn := &MockConn{}
	repo := NewServiceRepository(conn)
	ctx := context.Background()

	// Org A Query
	servicesA, err := repo.ListServices(ctx, OrgA)
	if err != nil {
		t.Fatalf("ListServices for OrgA failed: %v", err)
	}

	// Verify query string contains WHERE org_id = ?
	if !strings.Contains(conn.LastQuery, "WHERE org_id = ?") {
		t.Errorf("CRITICAL SECURITY LEAK: ListServices query missing WHERE org_id = ?: %s", conn.LastQuery)
	}
	if len(conn.LastArgs) == 0 || conn.LastArgs[0] != OrgA {
		t.Errorf("expected org_id arg %q, got %v", OrgA, conn.LastArgs)
	}

	// Assert Org A only sees Org A's services
	for _, s := range servicesA {
		if s == "secret-billing-svc" || s == "payroll-svc" {
			t.Fatalf("CROSS-TENANT LEAK: Org A saw Org B's private service: %s", s)
		}
	}

	// Org B Query
	servicesB, err := repo.ListServices(ctx, OrgB)
	if err != nil {
		t.Fatalf("ListServices for OrgB failed: %v", err)
	}

	for _, s := range servicesB {
		if s == "checkout-api" || s == "inventory-svc" {
			t.Fatalf("CROSS-TENANT LEAK: Org B saw Org A's service: %s", s)
		}
	}
}

func TestGetServiceGraph_TenantIsolation(t *testing.T) {
	conn := &MockConn{}
	repo := NewServiceRepository(conn)
	ctx := context.Background()

	graphA, err := repo.GetServiceGraph(ctx, OrgA, time.Now().Add(-1*time.Hour), time.Now())
	if err != nil {
		t.Fatalf("GetServiceGraph failed: %v", err)
	}

	// Assert SQL isolates both parent and child spans by org_id
	if !strings.Contains(conn.LastQuery, "org_id = ?") {
		t.Errorf("CRITICAL SECURITY LEAK: GetServiceGraph missing org_id filtering: %s", conn.LastQuery)
	}

	for _, node := range graphA.Nodes {
		if node.Name == "secret-billing-svc" || node.Name == "payroll-svc" {
			t.Fatalf("CROSS-TENANT LEAK: Org A service graph contains Org B node: %s", node.Name)
		}
	}
}

func TestSearchTraces_TenantIsolation(t *testing.T) {
	conn := &MockConn{}
	repo := NewTraceRepository(conn)
	ctx := context.Background()

	tracesA, err := repo.SearchTraces(ctx, OrgA, domain.TraceFilter{Limit: 10})
	if err != nil {
		t.Fatalf("SearchTraces failed: %v", err)
	}

	if !strings.Contains(conn.LastQuery, "org_id = ?") {
		t.Errorf("CRITICAL: SearchTraces query missing org_id = ?: %s", conn.LastQuery)
	}

	for _, tr := range tracesA {
		if tr.TraceID == "trace-beta-999" || tr.RootService == "secret-billing-svc" {
			t.Fatalf("CROSS-TENANT LEAK: Org A search returned Org B trace: %s", tr.TraceID)
		}
	}
}

func TestGetTraceWithSpans_TenantIsolation(t *testing.T) {
	conn := &MockConn{}
	repo := NewTraceRepository(conn)
	ctx := context.Background()

	// Org A requests its own trace -> returns trace
	trA, err := repo.GetTraceWithSpans(ctx, OrgA, "trace-alpha-001")
	if err != nil {
		t.Fatalf("expected successful lookup of own trace: %v", err)
	}
	if trA == nil || trA.TraceID != "trace-alpha-001" {
		t.Errorf("expected to find trace-alpha-001, got %v", trA)
	}

	// Org A maliciously attempts to fetch Org B's trace -> must return nil (not found)
	trB, err := repo.GetTraceWithSpans(ctx, OrgA, "trace-beta-999")
	if err != nil {
		t.Fatalf("unexpected error on cross-tenant lookup: %v", err)
	}
	if trB != nil {
		t.Fatalf("CROSS-TENANT DATA LEAK: Org A retrieved Org B's trace data: %+v", trB)
	}
}

func TestAnomalies_TenantIsolation(t *testing.T) {
	conn := &MockConn{}
	repo := NewAnomalyRepository(conn)
	ctx := context.Background()

	anomaliesA, err := repo.List(ctx, OrgA, domain.AnomalyFilter{Limit: 10})
	if err != nil {
		t.Fatalf("List anomalies failed: %v", err)
	}

	if !strings.Contains(conn.LastQuery, "WHERE org_id = ?") {
		t.Errorf("CRITICAL: Anomaly list missing WHERE org_id = ?: %s", conn.LastQuery)
	}

	for _, a := range anomaliesA {
		if a.OrgID != OrgA || a.ServiceName == "secret-billing-svc" {
			t.Fatalf("CROSS-TENANT LEAK: Org A received Org B anomaly: %+v", a)
		}
	}
}

func TestGetServiceStats_TenantIsolation(t *testing.T) {
	conn := &MockConn{}
	repo := NewServiceRepository(conn)
	ctx := context.Background()

	now := time.Now()
	stats, err := repo.GetServiceStats(ctx, OrgA, "checkout-api", now.Add(-1*time.Hour), now)
	if err != nil {
		t.Fatalf("GetServiceStats failed: %v", err)
	}

	if stats == nil || stats.ServiceName != "checkout-api" {
		t.Errorf("expected stats for checkout-api, got %+v", stats)
	}

	if !strings.Contains(conn.LastQuery, "WHERE org_id = ?") {
		t.Errorf("CRITICAL: GetServiceStats missing WHERE org_id = ?: %s", conn.LastQuery)
	}
}

func TestAnomaly_UpdateStatusAndStats(t *testing.T) {
	conn := &MockConn{}
	repo := NewAnomalyRepository(conn)
	ctx := context.Background()

	// 1. Update status
	if err := repo.UpdateStatus(ctx, OrgA, "anm-1", "resolved"); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}
	if !strings.Contains(conn.LastQuery, "WHERE id = ? AND org_id = ?") {
		t.Errorf("CRITICAL: UpdateStatus missing org_id check in query: %s", conn.LastQuery)
	}

	// 2. GetStats
	stats, err := repo.GetStats(ctx, OrgA)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats == nil || stats.Critical != 1 {
		t.Errorf("expected 1 critical anomaly in mock stats, got %+v", stats)
	}
}

func TestInsertSpansBatch(t *testing.T) {
	conn := &MockConn{}
	repo := NewTraceRepository(conn)
	ctx := context.Background()

	spans := []domain.Span{
		{
			OrgID:         OrgA,
			TraceID:       "trace-1",
			SpanID:        "span-1",
			ServiceName:   "checkout",
			OperationName: "POST /checkout",
			DurationMs:    100,
			StatusCode:    1,
			StartTime:     time.Now(),
		},
	}

	if err := repo.InsertSpansBatch(ctx, spans); err != nil {
		t.Fatalf("InsertSpansBatch failed: %v", err)
	}
}

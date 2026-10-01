package clickhouse

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
)

func TestGetServiceGraph_RealClickHouse_Integration(t *testing.T) {
	chURL := os.Getenv("CLICKHOUSE_URL")
	if chURL == "" {
		chURL = "127.0.0.1:9000"
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{chURL},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Skipf("skipping integration test: failed to open clickhouse connection: %v", err)
		return
	}

	var pingErr error
	for i := 0; i < 15; i++ {
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 1*time.Second)
		pingErr = conn.Ping(pingCtx)
		pingCancel()
		if pingErr == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if pingErr != nil {
		t.Skipf("skipping integration test: ClickHouse not reachable at %s (%v). Run 'docker compose up -d' to execute live integration tests.", chURL, pingErr)
		return
	}

	ctx := context.Background()

	// 1. Ensure test schema exists directly from the migration file (Single Source of Truth)
	migrationSQL, err := findMigrationFile("001_create_otel_spans.sql")
	if err != nil {
		t.Fatalf("failed to locate migration file: %v", err)
	}

	if err := conn.Exec(ctx, migrationSQL); err != nil {
		t.Fatalf("failed to execute migration 001_create_otel_spans.sql: %v", err)
	}

	anomalyMigrationSQL, err := findMigrationFile("002_create_anomalies.sql")
	if err != nil {
		t.Fatalf("failed to locate migration file: %v", err)
	}
	if err := conn.Exec(ctx, anomalyMigrationSQL); err != nil {
		t.Fatalf("failed to execute migration 002_create_anomalies.sql: %v", err)
	}

	repo := NewTraceRepository(conn)
	serviceRepo := NewServiceRepository(conn)
	anomalyRepo := NewAnomalyRepository(conn)

	now := time.Now().UTC()
	testOrgA := "integration-org-alpha"
	testOrgB := "integration-org-beta"

	// 2. Insert Real Parent-Child Spans for Org A
	spansA := []domain.Span{
		{
			OrgID:         testOrgA,
			TraceID:       "trace-integ-a-01",
			SpanID:        "span-a-root",
			ParentSpanID:  "",
			ServiceName:   "frontend-api-a",
			OperationName: "GET /checkout",
			DurationMs:    150,
			StatusCode:    1,
			StartTime:     now.Add(-5 * time.Minute),
		},
		{
			OrgID:         testOrgA,
			TraceID:       "trace-integ-a-01",
			SpanID:        "span-a-child",
			ParentSpanID:  "span-a-root",
			ServiceName:   "payment-svc-a",
			OperationName: "POST /charge",
			DurationMs:    100,
			StatusCode:    1,
			StartTime:     now.Add(-5 * time.Minute),
		},
	}

	// 3. Insert Real Parent-Child Spans for Org B
	spansB := []domain.Span{
		{
			OrgID:         testOrgB,
			TraceID:       "trace-integ-b-02",
			SpanID:        "span-b-root",
			ParentSpanID:  "",
			ServiceName:   "secret-bank-b",
			OperationName: "GET /wire",
			DurationMs:    300,
			StatusCode:    1,
			StartTime:     now.Add(-5 * time.Minute),
		},
		{
			OrgID:         testOrgB,
			TraceID:       "trace-integ-b-02",
			SpanID:        "span-b-child",
			ParentSpanID:  "span-b-root",
			ServiceName:   "vault-db-b",
			OperationName: "SELECT secret",
			DurationMs:    200,
			StatusCode:    1,
			StartTime:     now.Add(-5 * time.Minute),
		},
	}

	if err := repo.InsertSpansBatch(ctx, append(spansA, spansB...)); err != nil {
		t.Fatalf("failed to insert test spans: %v", err)
	}

	// Insert an active critical anomaly for payment-svc-a in Org A
	anmA := &domain.Anomaly{
		ID:                "anm-integ-a-01",
		OrgID:             testOrgA,
		ServiceName:       "payment-svc-a",
		OperationName:     "POST /charge",
		RootCauseService:  "payment-svc-a",
		Severity:          "critical",
		ZScore:            4.8,
		BaselineLatencyMs: 15.0,
		ObservedLatencyMs: 100.0,
		Status:            "detected",
		DetectedAt:        now,
	}
	if err := anomalyRepo.Insert(ctx, anmA); err != nil {
		t.Fatalf("failed to insert anomaly for Org A: %v", err)
	}

	// 4. Query GetServiceGraph for Org A
	from := now.Add(-1 * time.Hour)
	to := now.Add(1 * time.Hour)

	graphA, err := serviceRepo.GetServiceGraph(ctx, testOrgA, from, to)
	if err != nil {
		t.Fatalf("GetServiceGraph failed on live ClickHouse: %v", err)
	}

	// 5. Assert Tenant Isolation on Real SQL JOIN Execution
	for _, node := range graphA.Nodes {
		if node.Name == "secret-bank-b" || node.Name == "vault-db-b" {
			t.Fatalf("LIVE CLICKHOUSE LEAK: Org A service graph returned Org B node: %s", node.Name)
		}
	}

	for _, edge := range graphA.Edges {
		if edge.Source == "secret-bank-b" || edge.Target == "vault-db-b" {
			t.Fatalf("LIVE CLICKHOUSE LEAK: Org A service graph returned Org B edge: %+v", edge)
		}
	}

	// Assert Org A's own edge was discovered
	foundEdge := false
	for _, edge := range graphA.Edges {
		if edge.Source == "frontend-api-a" && edge.Target == "payment-svc-a" {
			foundEdge = true
			break
		}
	}
	if !foundEdge && len(graphA.Edges) > 0 {
		t.Errorf("expected to find edge frontend-api-a -> payment-svc-a in graph: %+v", graphA.Edges)
	}

	// Assert payment-svc-a health is 'critical' driven by the active anomaly
	var paymentNode *domain.ServiceNode
	for i := range graphA.Nodes {
		if graphA.Nodes[i].Name == "payment-svc-a" {
			paymentNode = &graphA.Nodes[i]
			break
		}
	}
	if paymentNode == nil {
		t.Fatal("expected payment-svc-a node in Org A graph")
	}
	if paymentNode.Health != "critical" {
		t.Errorf("expected payment-svc-a health to be 'critical' driven by active anomaly, got %s", paymentNode.Health)
	}
	if paymentNode.ActiveAnomalies != 1 {
		t.Errorf("expected payment-svc-a to have 1 active anomaly, got %d", paymentNode.ActiveAnomalies)
	}
}

func findMigrationFile(filename string) (string, error) {
	candidates := []string{
		filepath.Join("..", "..", "..", "migrations", "clickhouse", filename),
		filepath.Join("migrations", "clickhouse", filename),
	}
	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			return string(data), nil
		}
	}
	return "", fmt.Errorf("migration file %s not found in candidate paths %v", filename, candidates)
}


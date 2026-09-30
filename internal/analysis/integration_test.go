package analysis

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	chRepo "github.com/RaviBharathi410/distributedtrace/internal/repository/clickhouse"
)

func isPortReachable(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func findMigrationFile(name string) (string, error) {
	paths := []string{
		filepath.Join("..", "..", "migrations", "clickhouse", name),
		filepath.Join("migrations", "clickhouse", name),
	}
	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil {
			return string(data), nil
		}
	}
	return "", fmt.Errorf("migration file %s not found in candidates %v", name, paths)
}

func TestAnomalyDetector_RealClickHouse_EndToEndCausalAnalysis(t *testing.T) {
	chURL := os.Getenv("CLICKHOUSE_URL")
	if chURL == "" {
		chURL = "127.0.0.1:9000"
	}

	if !isPortReachable(chURL) {
		t.Skipf("skipping live ClickHouse anomaly detection integration test: ClickHouse not reachable at %s. Run 'docker compose up -d' or execute in CI.", chURL)
		return
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

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := conn.Ping(ctx); err != nil {
		t.Skipf("skipping integration test: ClickHouse ping failed: %v", err)
		return
	}

	// 1. Ensure migrations 001 and 002 exist on live ClickHouse
	spansDDL, err := findMigrationFile("001_create_otel_spans.sql")
	if err != nil {
		t.Fatalf("failed to locate 001_create_otel_spans.sql: %v", err)
	}
	if err := conn.Exec(ctx, spansDDL); err != nil {
		t.Fatalf("failed to execute 001_create_otel_spans.sql: %v", err)
	}

	anomDDL, err := findMigrationFile("002_create_anomalies.sql")
	if err != nil {
		t.Fatalf("failed to locate 002_create_anomalies.sql: %v", err)
	}
	if err := conn.Exec(ctx, anomDDL); err != nil {
		t.Fatalf("failed to execute 002_create_anomalies.sql: %v", err)
	}

	traceRepo := chRepo.NewTraceRepository(conn)
	anomalyRepo := chRepo.NewAnomalyRepository(conn)

	baselineCalc := NewBaselineCalculator()
	causalAnalyzer := NewCausalAnalyzer()
	detector := NewAnomalyDetector(baselineCalc, causalAnalyzer, anomalyRepo)

	nanos := time.Now().UnixNano()
	testOrgA := fmt.Sprintf("org-causal-a-%d", nanos)
	testOrgB := fmt.Sprintf("org-causal-b-%d", nanos)
	traceID := fmt.Sprintf("trace-anom-%d", nanos)
	now := time.Now().UTC()

	// 2. Pre-train baseline for Org A's web-frontend: 70ms normal duration
	for i := 0; i < 10; i++ {
		baselineCalc.Update(testOrgA, "web-frontend", "GET /checkout", 70.0)
	}

	// 3. Construct 3-tier distributed trace with downstream bottleneck:
	// web-frontend (1850ms) -> cart-service (1820ms) -> pricing-db (1800ms)
	spans := []domain.Span{
		{
			OrgID:         testOrgA,
			TraceID:       traceID,
			SpanID:        "span-front",
			ParentSpanID:  "",
			ServiceName:   "web-frontend",
			OperationName: "GET /checkout",
			DurationMs:    1850,
			StatusCode:    1,
			StartTime:     now,
		},
		{
			OrgID:         testOrgA,
			TraceID:       traceID,
			SpanID:        "span-cart",
			ParentSpanID:  "span-front",
			ServiceName:   "cart-service",
			OperationName: "CalculateTotals",
			DurationMs:    1820,
			StatusCode:    1,
			StartTime:     now.Add(10 * time.Millisecond),
		},
		{
			OrgID:         testOrgA,
			TraceID:       traceID,
			SpanID:        "span-db",
			ParentSpanID:  "span-cart",
			ServiceName:   "pricing-db",
			OperationName: "SELECT prices_bulk",
			DurationMs:    1800,
			StatusCode:    1,
			StartTime:     now.Add(20 * time.Millisecond),
		},
	}

	// 4. Persist spans to real ClickHouse otel_spans table
	if err := traceRepo.InsertSpansBatch(ctx, spans); err != nil {
		t.Fatalf("failed to insert test spans to ClickHouse: %v", err)
	}

	// 5. Run AnomalyDetector on the trace
	detected, err := detector.AnalyzeTrace(ctx, testOrgA, spans)
	if err != nil {
		t.Fatalf("AnalyzeTrace failed: %v", err)
	}
	if detected == nil {
		t.Fatalf("expected anomaly to be detected for 1850ms trace (baseline 70ms)")
	}

	// 6. Assert Causal Analysis Results
	if detected.RootCauseService != "pricing-db" {
		t.Fatalf("expected RootCauseService 'pricing-db', got %q", detected.RootCauseService)
	}
	if detected.Severity != domain.SeverityCritical {
		t.Fatalf("expected Severity 'critical', got %q", detected.Severity)
	}
	expectedPath := []string{"web-frontend", "cart-service", "pricing-db"}
	if len(detected.RootCausePath) != 3 || detected.RootCausePath[2] != "pricing-db" {
		t.Fatalf("expected RootCausePath %+v, got %+v", expectedPath, detected.RootCausePath)
	}

	// 7. Verify ClickHouse Persistence & Read Path
	anomaliesA, err := anomalyRepo.List(ctx, testOrgA, domain.AnomalyFilter{Limit: 10})
	if err != nil {
		t.Fatalf("failed to query anomalies from ClickHouse: %v", err)
	}
	if len(anomaliesA) != 1 {
		t.Fatalf("expected exactly 1 anomaly in ClickHouse for Org A, got %d", len(anomaliesA))
	}
	if anomaliesA[0].RootCauseService != "pricing-db" {
		t.Errorf("ClickHouse stored anomaly root_cause_service mismatch: expected 'pricing-db', got %q", anomaliesA[0].RootCauseService)
	}

	statsA, err := anomalyRepo.GetStats(ctx, testOrgA)
	if err != nil {
		t.Fatalf("failed to query anomaly stats from ClickHouse: %v", err)
	}
	if statsA.Critical < 1 {
		t.Errorf("expected at least 1 critical anomaly in stats, got %+v", statsA)
	}

	// 8. Strict Multi-Tenant Isolation: Org B must not see Org A's anomalies
	anomaliesB, err := anomalyRepo.List(ctx, testOrgB, domain.AnomalyFilter{Limit: 10})
	if err != nil {
		t.Fatalf("failed to query Org B anomalies: %v", err)
	}
	if len(anomaliesB) != 0 {
		t.Fatalf("TENANT LEAK: Org B received Org A anomaly: %+v", anomaliesB)
	}

	statsB, err := anomalyRepo.GetStats(ctx, testOrgB)
	if err != nil {
		t.Fatalf("failed to query Org B stats: %v", err)
	}
	if statsB.Critical != 0 || statsB.High != 0 || statsB.Medium != 0 || statsB.Low != 0 {
		t.Fatalf("TENANT LEAK: Org B stats returned non-zero count for Org A anomaly: %+v", statsB)
	}
}

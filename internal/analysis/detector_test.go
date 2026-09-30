package analysis

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
)

type mockAnomalyWriter struct {
	mu        sync.Mutex
	anomalies []*domain.Anomaly
}

func (m *mockAnomalyWriter) Insert(ctx context.Context, a *domain.Anomaly) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.anomalies = append(m.anomalies, a)
	return nil
}

func (m *mockAnomalyWriter) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.anomalies)
}

func TestAnomalyDetector_DetectionAndPersistence(t *testing.T) {
	ctx := context.Background()
	writer := &mockAnomalyWriter{}
	baseline := NewBaselineCalculator()
	causal := NewCausalAnalyzer()
	detector := NewAnomalyDetector(baseline, causal, writer)

	orgID := "org-acme"
	service := "checkout-service"
	op := "POST /checkout"

	// 1. Train baseline with 10 normal traces (average 80ms, small variance)
	for i := 0; i < 10; i++ {
		baseline.Update(orgID, service, op, 80.0)
	}

	// 2. Normal Trace: root span takes 82ms -> should NOT trigger anomaly
	normalTrace := []domain.Span{
		{
			OrgID:         orgID,
			TraceID:       "trace-normal-1",
			SpanID:        "span-root",
			ParentSpanID:  "",
			ServiceName:   service,
			OperationName: op,
			DurationMs:    82,
			StartTime:     time.Now().UTC(),
		},
	}

	anom, err := detector.AnalyzeTrace(ctx, orgID, normalTrace)
	if err != nil {
		t.Fatalf("unexpected error analyzing normal trace: %v", err)
	}
	if anom != nil {
		t.Fatalf("expected nil anomaly for normal trace, got %+v", anom)
	}
	if writer.Count() != 0 {
		t.Fatalf("expected 0 anomalies persisted for normal trace, got %d", writer.Count())
	}

	// 3. Anomalous Trace: downstream database spike causes root to inflate from 80ms -> 600ms
	now := time.Now().UTC()
	anomalousTrace := []domain.Span{
		{
			OrgID:         orgID,
			TraceID:       "trace-spike-1",
			SpanID:        "span-root",
			ParentSpanID:  "",
			ServiceName:   service,
			OperationName: op,
			DurationMs:    600,
			StartTime:     now,
		},
		{
			OrgID:         orgID,
			TraceID:       "trace-spike-1",
			SpanID:        "span-pay",
			ParentSpanID:  "span-root",
			ServiceName:   "payment-service",
			OperationName: "ProcessPayment",
			DurationMs:    550,
			StartTime:     now.Add(10 * time.Millisecond),
		},
		{
			OrgID:         orgID,
			TraceID:       "trace-spike-1",
			SpanID:        "span-db",
			ParentSpanID:  "span-pay",
			ServiceName:   "postgres-db",
			OperationName: "UPDATE balances",
			DurationMs:    520,
			StartTime:     now.Add(20 * time.Millisecond),
		},
	}

	anom, err = detector.AnalyzeTrace(ctx, orgID, anomalousTrace)
	if err != nil {
		t.Fatalf("unexpected error analyzing anomalous trace: %v", err)
	}
	if anom == nil {
		t.Fatalf("expected anomaly to be detected for 600ms trace")
	}

	// Validate anomaly payload
	if anom.OrgID != orgID {
		t.Errorf("expected OrgID %q, got %q", orgID, anom.OrgID)
	}
	if anom.ServiceName != service {
		t.Errorf("expected ServiceName %q, got %q", service, anom.ServiceName)
	}
	if anom.RootCauseService != "postgres-db" {
		t.Errorf("expected RootCauseService 'postgres-db', got %q", anom.RootCauseService)
	}
	if anom.Severity != domain.SeverityCritical {
		t.Errorf("expected SeverityCritical for large Z-score, got %q (Z=%.2f)", anom.Severity, anom.ZScore)
	}
	if anom.BaselineLatencyMs != 80.0 {
		t.Errorf("expected BaselineLatencyMs 80.0, got %.2f", anom.BaselineLatencyMs)
	}
	if anom.ObservedLatencyMs != 600.0 {
		t.Errorf("expected ObservedLatencyMs 600.0, got %.2f", anom.ObservedLatencyMs)
	}

	expectedPath := []string{"checkout-service", "payment-service", "postgres-db"}
	if len(anom.RootCausePath) != 3 || anom.RootCausePath[2] != "postgres-db" {
		t.Errorf("expected RootCausePath %+v, got %+v", expectedPath, anom.RootCausePath)
	}

	// Assert persistence in writer
	if writer.Count() != 1 {
		t.Fatalf("expected 1 anomaly persisted in writer, got %d", writer.Count())
	}
}

func TestAnomalyDetector_ProcessBatch(t *testing.T) {
	ctx := context.Background()
	writer := &mockAnomalyWriter{}
	baseline := NewBaselineCalculator()
	detector := NewAnomalyDetector(baseline, nil, writer)

	orgID := "org-batch"
	service := "search-service"
	op := "GET /search"

	// Train baseline
	for i := 0; i < 10; i++ {
		baseline.Update(orgID, service, op, 50.0)
	}

	batch := []domain.Span{
		// Trace 1: Normal (52ms)
		{
			OrgID:         orgID,
			TraceID:       "trace-1",
			SpanID:        "span-1",
			ServiceName:   service,
			OperationName: op,
			DurationMs:    52,
			StartTime:     time.Now().UTC(),
		},
		// Trace 2: Anomalous (350ms)
		{
			OrgID:         orgID,
			TraceID:       "trace-2",
			SpanID:        "span-2",
			ServiceName:   service,
			OperationName: op,
			DurationMs:    350,
			StartTime:     time.Now().UTC(),
		},
	}

	detected, err := detector.ProcessBatch(ctx, batch)
	if err != nil {
		t.Fatalf("ProcessBatch failed: %v", err)
	}

	if len(detected) != 1 {
		t.Fatalf("expected exactly 1 anomaly detected in batch, got %d", len(detected))
	}
	if detected[0].ObservedLatencyMs != 350.0 {
		t.Errorf("expected observed latency 350.0, got %.2f", detected[0].ObservedLatencyMs)
	}
}

package analysis

import (
	"reflect"
	"testing"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
)

func TestCausalAnalyzer_DeepChainDownstreamRootCause(t *testing.T) {
	now := time.Now().UTC()
	traceID := "trace-causal-chain-1"

	// 4-Tier Call Chain:
	// api-gateway (2100ms total, 50ms self)
	//   -> order-service (2050ms total, 30ms self)
	//     -> payment-service (2020ms total, 20ms self)
	//       -> payment-db (2000ms total, 2000ms self)
	spans := []domain.Span{
		{
			TraceID:       traceID,
			SpanID:        "span-root",
			ParentSpanID:  "",
			ServiceName:   "api-gateway",
			OperationName: "POST /order",
			DurationMs:    2100,
			StartTime:     now,
		},
		{
			TraceID:       traceID,
			SpanID:        "span-order",
			ParentSpanID:  "span-root",
			ServiceName:   "order-service",
			OperationName: "ProcessOrder",
			DurationMs:    2050,
			StartTime:     now.Add(10 * time.Millisecond),
		},
		{
			TraceID:       traceID,
			SpanID:        "span-payment",
			ParentSpanID:  "span-order",
			ServiceName:   "payment-service",
			OperationName: "ChargeCard",
			DurationMs:    2020,
			StartTime:     now.Add(20 * time.Millisecond),
		},
		{
			TraceID:       traceID,
			SpanID:        "span-db",
			ParentSpanID:  "span-payment",
			ServiceName:   "payment-db",
			OperationName: "INSERT transactions",
			DurationMs:    2000,
			StartTime:     now.Add(30 * time.Millisecond),
		},
	}

	analyzer := NewCausalAnalyzer()
	res := analyzer.AnalyzeRootCause(spans)

	if res.RootCauseService != "payment-db" {
		t.Fatalf("expected RootCauseService 'payment-db', got %q", res.RootCauseService)
	}

	expectedPath := []string{"api-gateway", "order-service", "payment-service", "payment-db"}
	if !reflect.DeepEqual(res.RootCausePath, expectedPath) {
		t.Fatalf("expected RootCausePath %+v, got %+v", expectedPath, res.RootCausePath)
	}

	if res.RootCauseSpanID != "span-db" {
		t.Errorf("expected RootCauseSpanID 'span-db', got %q", res.RootCauseSpanID)
	}
}

func TestCausalAnalyzer_SiblingBranchDiscrimination(t *testing.T) {
	now := time.Now().UTC()
	traceID := "trace-causal-siblings"

	// api-gateway calls fast auth-service (15ms) and slow warehouse-service (1500ms)
	spans := []domain.Span{
		{
			TraceID:       traceID,
			SpanID:        "span-gw",
			ParentSpanID:  "",
			ServiceName:   "api-gateway",
			OperationName: "GET /inventory",
			DurationMs:    1550,
			StartTime:     now,
		},
		{
			TraceID:       traceID,
			SpanID:        "span-auth",
			ParentSpanID:  "span-gw",
			ServiceName:   "auth-service",
			OperationName: "VerifyToken",
			DurationMs:    15,
			StartTime:     now.Add(5 * time.Millisecond),
		},
		{
			TraceID:       traceID,
			SpanID:        "span-wh",
			ParentSpanID:  "span-gw",
			ServiceName:   "warehouse-service",
			OperationName: "QueryStock",
			DurationMs:    1500,
			StartTime:     now.Add(25 * time.Millisecond),
		},
	}

	analyzer := NewCausalAnalyzer()
	res := analyzer.AnalyzeRootCause(spans)

	if res.RootCauseService != "warehouse-service" {
		t.Fatalf("expected bottleneck branch 'warehouse-service', got %q", res.RootCauseService)
	}

	expectedPath := []string{"api-gateway", "warehouse-service"}
	if !reflect.DeepEqual(res.RootCausePath, expectedPath) {
		t.Fatalf("expected RootCausePath %+v, got %+v", expectedPath, res.RootCausePath)
	}
}

func TestCausalAnalyzer_RootNodeBottleneck(t *testing.T) {
	now := time.Now().UTC()
	traceID := "trace-root-bottleneck"

	// api-gateway takes 1000ms doing heavy CPU JSON rendering, child takes only 20ms
	spans := []domain.Span{
		{
			TraceID:       traceID,
			SpanID:        "span-gw",
			ParentSpanID:  "",
			ServiceName:   "api-gateway",
			OperationName: "GET /export",
			DurationMs:    1000,
			StartTime:     now,
		},
		{
			TraceID:       traceID,
			SpanID:        "span-child",
			ParentSpanID:  "span-gw",
			ServiceName:   "db-service",
			OperationName: "QuickSelect",
			DurationMs:    20,
			StartTime:     now.Add(5 * time.Millisecond),
		},
	}

	analyzer := NewCausalAnalyzer()
	res := analyzer.AnalyzeRootCause(spans)

	if res.RootCauseService != "api-gateway" {
		t.Fatalf("expected root service 'api-gateway' as root cause when self-time dominates, got %q", res.RootCauseService)
	}

	expectedPath := []string{"api-gateway"}
	if !reflect.DeepEqual(res.RootCausePath, expectedPath) {
		t.Fatalf("expected RootCausePath %+v, got %+v", expectedPath, res.RootCausePath)
	}
}

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/auth"
	chRepo "github.com/RaviBharathi410/distributedtrace/internal/repository/clickhouse"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	trace "go.opentelemetry.io/otel/trace"
)

func TestRealOpenTelemetrySDK_IngestOTLP_ProtobufGzip(t *testing.T) {
	mockConn := &apiMockConn{}
	traceRepo := chRepo.NewTraceRepository(mockConn)
	spanHandler := NewSpanHandler(traceRepo, nil)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Mock authentication: bind TestOrgA into request context
			ctx := context.WithValue(req.Context(), auth.OrgIDKey, TestOrgA)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Post("/v1/traces", spanHandler.IngestOTLP)

	server := httptest.NewServer(r)
	defer server.Close()

	ctx := context.Background()

	// Initialize REAL official OpenTelemetry Go SDK HTTP exporter
	// Defaults to Protobuf wire format and Gzip compression
	endpoint := server.Listener.Addr().String()
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
	)
	if err != nil {
		t.Fatalf("failed to create real OTLP HTTP exporter: %v", err)
	}
	defer exporter.Shutdown(ctx)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
	)
	defer tp.Shutdown(ctx)

	tracer := tp.Tracer("real-otel-go-sdk-tracer")

	// Emit a real distributed trace with parent and child span
	ctx, parentSpan := tracer.Start(ctx, "checkout.process_order",
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("http.method", "POST"),
			attribute.Int("http.status_code", 201),
			semconv.ServiceName("checkout-service"),
		),
	)
	time.Sleep(10 * time.Millisecond)

	_, childSpan := tracer.Start(ctx, "db.query_inventory",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "postgres"),
			attribute.String("db.statement", "SELECT quantity FROM items WHERE item_id = 42"),
		),
	)
	time.Sleep(5 * time.Millisecond)
	childSpan.End()

	parentSpan.End()

	// Force flush all spans through the SDK pipeline to our server
	flushCtx, flushCancel := context.WithTimeout(ctx, 5*time.Second)
	defer flushCancel()
	if err := tp.ForceFlush(flushCtx); err != nil {
		t.Fatalf("failed to force flush spans from SDK: %v", err)
	}

	// Verify ClickHouse batch persistence
	if len(mockConn.batches) == 0 {
		t.Fatalf("expected ClickHouse batch to be prepared")
	}

	totalSpans := 0
	for _, b := range mockConn.batches {
		totalSpans += len(b.appendedRows)
	}

	if totalSpans < 2 {
		t.Fatalf("expected at least 2 spans ingested from real SDK, got %d", totalSpans)
	}

	t.Logf("Successfully received and parsed %d spans from REAL OpenTelemetry Go SDK over Protobuf+Gzip!", totalSpans)
	firstRow := mockConn.batches[0].appendedRows[0]
	orgID := firstRow[0].(string)
	if orgID != TestOrgA {
		t.Errorf("expected tenant org %s, got %s", TestOrgA, orgID)
	}
}

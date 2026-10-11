package ingest

import (
	"testing"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

func TestParseOTLPTracesJSON_ValidPayload(t *testing.T) {
	otlpJSON := `{
		"resourceSpans": [
			{
				"resource": {
					"attributes": [
						{"key": "service.name", "value": {"stringValue": "checkout-api"}},
						{"key": "deployment.environment", "value": {"stringValue": "production"}}
					]
				},
				"scopeSpans": [
					{
						"scope": {"name": "order-processor", "version": "1.2.0"},
						"spans": [
							{
								"traceId": "4bf92f3577b34da6a3ce929d0e0e4736",
								"spanId": "00f067aa0ba902b7",
								"name": "POST /order",
								"kind": 2,
								"startTimeUnixNano": "1728662400000000000",
								"endTimeUnixNano": "1728662400150000000",
								"attributes": [
									{"key": "http.status_code", "value": {"intValue": 200}},
									{"key": "http.method", "value": {"stringValue": "POST"}}
								],
								"status": {
									"code": 1,
									"message": "success"
								}
							},
							{
								"traceId": "4bf92f3577b34da6a3ce929d0e0e4736",
								"spanId": "5fb397be34d23b72",
								"parentSpanId": "00f067aa0ba902b7",
								"name": "SELECT customer",
								"kind": 3,
								"startTimeUnixNano": 1728662400020000000,
								"endTimeUnixNano": 1728662400070000000,
								"attributes": [
									{"key": "db.system", "value": {"stringValue": "postgresql"}}
								],
								"status": {
									"code": 2,
									"message": "connection pool timeout"
								}
							}
						]
					}
				]
			}
		]
	}`

	spans, err := ParseOTLPTracesJSON([]byte(otlpJSON))
	if err != nil {
		t.Fatalf("unexpected error parsing OTLP JSON: %v", err)
	}

	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}

	s1 := spans[0]
	if s1.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("expected TraceID '4bf92f3577b34da6a3ce929d0e0e4736', got %q", s1.TraceID)
	}
	if s1.SpanID != "00f067aa0ba902b7" {
		t.Errorf("expected SpanID '00f067aa0ba902b7', got %q", s1.SpanID)
	}
	if s1.ServiceName != "checkout-api" {
		t.Errorf("expected ServiceName 'checkout-api', got %q", s1.ServiceName)
	}
	if s1.OperationName != "POST /order" {
		t.Errorf("expected OperationName 'POST /order', got %q", s1.OperationName)
	}
	if s1.DurationMs != 150 {
		t.Errorf("expected DurationMs 150, got %d", s1.DurationMs)
	}
	if s1.StatusCode != 1 {
		t.Errorf("expected StatusCode 1, got %d", s1.StatusCode)
	}
	if s1.Tags["deployment.environment"] != "production" {
		t.Errorf("expected resource tag deployment.environment=production, got %q", s1.Tags["deployment.environment"])
	}
	if s1.Tags["http.status_code"] != "200" {
		t.Errorf("expected span tag http.status_code=200, got %q", s1.Tags["http.status_code"])
	}

	s2 := spans[1]
	if s2.ParentSpanID != "00f067aa0ba902b7" {
		t.Errorf("expected ParentSpanID '00f067aa0ba902b7', got %q", s2.ParentSpanID)
	}
	if s2.DurationMs != 50 {
		t.Errorf("expected DurationMs 50, got %d", s2.DurationMs)
	}
	if s2.StatusCode != 2 {
		t.Errorf("expected StatusCode 2, got %d", s2.StatusCode)
	}
	if s2.ErrorMessage != "connection pool timeout" {
		t.Errorf("expected ErrorMessage 'connection pool timeout', got %q", s2.ErrorMessage)
	}

	// Verify validation & enrichment succeed on the parsed spans
	if err := ValidateSpans(spans); err != nil {
		t.Fatalf("validation failed on parsed OTLP spans: %v", err)
	}
	enriched := EnrichSpans(spans, "org-test-123")
	if enriched[0].OrgID != "org-test-123" {
		t.Errorf("expected OrgID 'org-test-123', got %q", enriched[0].OrgID)
	}
}

func TestParseOTLPTracesJSON_SubMillisecondDuration(t *testing.T) {
	otlpJSON := `{
		"resourceSpans": [
			{
				"resource": {"attributes": [{"key": "service.name", "value": {"stringValue": "cache-svc"}}]},
				"scopeSpans": [
					{
						"spans": [
							{
								"traceId": "trace-sub-ms",
								"spanId": "span-sub-ms",
								"name": "GET /cache",
								"startTimeUnixNano": "1728662400000000000",
								"endTimeUnixNano": "1728662400000500000"
							}
						]
					}
				]
			}
		]
	}`

	spans, err := ParseOTLPTracesJSON([]byte(otlpJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].DurationMs != 1 {
		t.Errorf("expected sub-millisecond span to floor at 1ms, got %d", spans[0].DurationMs)
	}
}

func TestParseOTLPTracesProto_ValidPayload(t *testing.T) {
	traceIDBytes := []byte{0x4b, 0xf9, 0x2f, 0x35, 0x77, 0xb3, 0x4d, 0xa6, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36}
	spanIDBytes := []byte{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7}
	parentSpanIDBytes := []byte{0x5f, 0xb3, 0x97, 0xbe, 0x34, 0xd2, 0x3b, 0x72}

	protoReq := &coltracepb.ExportTraceServiceRequest{
		ResourceSpans: []*tracepb.ResourceSpans{
			{
				Resource: &resourcepb.Resource{
					Attributes: []*commonpb.KeyValue{
						{
							Key:   "service.name",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "payment-svc"}},
						},
						{
							Key:   "k8s.cluster",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "prod-us-east"}},
						},
					},
				},
				ScopeSpans: []*tracepb.ScopeSpans{
					{
						Spans: []*tracepb.Span{
							{
								TraceId:           traceIDBytes,
								SpanId:            spanIDBytes,
								ParentSpanId:      parentSpanIDBytes,
								Name:              "POST /charge",
								Kind:              tracepb.Span_SPAN_KIND_SERVER,
								StartTimeUnixNano: 1728662400000000000,
								EndTimeUnixNano:   1728662400085000000,
								Attributes: []*commonpb.KeyValue{
									{
										Key:   "http.status_code",
										Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 500}},
									},
								},
								Status: &tracepb.Status{
									Code:    tracepb.Status_STATUS_CODE_ERROR,
									Message: "database connection timeout",
								},
							},
						},
					},
				},
			},
		},
	}

	rawBytes, err := proto.Marshal(protoReq)
	if err != nil {
		t.Fatalf("failed to marshal proto req: %v", err)
	}

	spans, err := ParseOTLPTracesProto(rawBytes)
	if err != nil {
		t.Fatalf("unexpected error parsing protobuf: %v", err)
	}

	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	s := spans[0]
	expectedTraceHex := "4bf92f3577b34da6a3ce929d0e0e4736"
	expectedSpanHex := "00f067aa0ba902b7"
	expectedParentHex := "5fb397be34d23b72"

	if s.TraceID != expectedTraceHex {
		t.Errorf("expected TraceID %s, got %s", expectedTraceHex, s.TraceID)
	}
	if s.SpanID != expectedSpanHex {
		t.Errorf("expected SpanID %s, got %s", expectedSpanHex, s.SpanID)
	}
	if s.ParentSpanID != expectedParentHex {
		t.Errorf("expected ParentSpanID %s, got %s", expectedParentHex, s.ParentSpanID)
	}
	if s.ServiceName != "payment-svc" {
		t.Errorf("expected ServiceName 'payment-svc', got %s", s.ServiceName)
	}
	if s.OperationName != "POST /charge" {
		t.Errorf("expected OperationName 'POST /charge', got %s", s.OperationName)
	}
	if s.DurationMs != 85 {
		t.Errorf("expected DurationMs 85, got %d", s.DurationMs)
	}
	if s.StatusCode != 2 {
		t.Errorf("expected StatusCode 2, got %d", s.StatusCode)
	}
	if s.ErrorMessage != "database connection timeout" {
		t.Errorf("expected ErrorMessage 'database connection timeout', got %q", s.ErrorMessage)
	}
	if s.Tags["k8s.cluster"] != "prod-us-east" {
		t.Errorf("expected tag k8s.cluster=prod-us-east, got %q", s.Tags["k8s.cluster"])
	}
	if s.Tags["http.status_code"] != "500" {
		t.Errorf("expected tag http.status_code=500, got %q", s.Tags["http.status_code"])
	}

	// Verify BuildOTLPProtoSuccessResponse
	respBytes, err := BuildOTLPProtoSuccessResponse()
	if err != nil {
		t.Fatalf("failed to build proto success response: %v", err)
	}
	var resp coltracepb.ExportTraceServiceResponse
	if err := proto.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("failed to unmarshal proto success response: %v", err)
	}
}

func TestEnrichSpans_TenantIsolationAndSpoofing(t *testing.T) {
	spans := []domain.Span{
		{
			TraceID:       "trace-1",
			SpanID:        "span-1",
			ServiceName:   "order-svc",
			OperationName: "POST /order",
			OrgID:         "malicious-attacker-org",
			Tags: map[string]string{
				"org_id":    "spoofed-org-666",
				"tenant_id": "spoofed-tenant-777",
				"env":       "prod",
			},
		},
	}

	authenticOrg := "legitimate-org-123"
	enriched := EnrichSpans(spans, authenticOrg)

	if len(enriched) != 1 {
		t.Fatalf("expected 1 enriched span")
	}
	e := enriched[0]
	if e.OrgID != authenticOrg {
		t.Errorf("expected OrgID to be overwritten with %s, got %s", authenticOrg, e.OrgID)
	}
	if e.Tags["org_id"] != authenticOrg {
		t.Errorf("expected Tags[org_id] to be clamped to %s, got %s", authenticOrg, e.Tags["org_id"])
	}
	if _, exists := e.Tags["tenant_id"]; exists {
		t.Errorf("expected Tags[tenant_id] to be purged to prevent spoofing")
	}
	if e.Tags["env"] != "prod" {
		t.Errorf("expected ordinary tag env=prod to be preserved, got %s", e.Tags["env"])
	}
}

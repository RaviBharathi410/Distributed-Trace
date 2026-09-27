package ingest

import (
	"testing"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
)

func TestValidateSpans(t *testing.T) {
	now := time.Now()

	t.Run("Valid spans pass validation", func(t *testing.T) {
		spans := []domain.Span{
			{
				TraceID:       "4bf92f3577b34da6a3ce929d0e0e4736",
				SpanID:        "00f067aa0ba902b7",
				ServiceName:   "checkout-api",
				OperationName: "POST /checkout",
				StartTime:     now,
				DurationMs:    120,
			},
		}

		if err := ValidateSpans(spans); err != nil {
			t.Errorf("expected valid spans to pass, got: %v", err)
		}
	})

	t.Run("Fails when batch is empty", func(t *testing.T) {
		if err := ValidateSpans(nil); err == nil {
			t.Errorf("expected error on empty batch")
		}
	})

	t.Run("Fails when batch exceeds 1000 spans", func(t *testing.T) {
		hugeBatch := make([]domain.Span, 1001)
		if err := ValidateSpans(hugeBatch); err == nil {
			t.Errorf("expected error on batch > 1000")
		}
	})

	t.Run("Fails when trace_id is missing", func(t *testing.T) {
		spans := []domain.Span{
			{
				SpanID:        "00f067aa0ba902b7",
				ServiceName:   "checkout-api",
				OperationName: "POST /checkout",
				StartTime:     now,
			},
		}
		if err := ValidateSpans(spans); err == nil {
			t.Errorf("expected error when trace_id is missing")
		}
	})

	t.Run("Fails when span_id is missing", func(t *testing.T) {
		spans := []domain.Span{
			{
				TraceID:       "4bf92f3577b34da6a3ce929d0e0e4736",
				ServiceName:   "checkout-api",
				OperationName: "POST /checkout",
				StartTime:     now,
			},
		}
		if err := ValidateSpans(spans); err == nil {
			t.Errorf("expected error when span_id is missing")
		}
	})

	t.Run("Fails when service_name is missing", func(t *testing.T) {
		spans := []domain.Span{
			{
				TraceID:       "4bf92f3577b34da6a3ce929d0e0e4736",
				SpanID:        "00f067aa0ba902b7",
				OperationName: "POST /checkout",
				StartTime:     now,
			},
		}
		if err := ValidateSpans(spans); err == nil {
			t.Errorf("expected error when service_name is missing")
		}
	})

	t.Run("Fails when start_time is zero", func(t *testing.T) {
		spans := []domain.Span{
			{
				TraceID:       "4bf92f3577b34da6a3ce929d0e0e4736",
				SpanID:        "00f067aa0ba902b7",
				ServiceName:   "checkout-api",
				OperationName: "POST /checkout",
			},
		}
		if err := ValidateSpans(spans); err == nil {
			t.Errorf("expected error when start_time is zero")
		}
	})

	t.Run("Fails when timestamp is in future", func(t *testing.T) {
		spans := []domain.Span{
			{
				TraceID:       "4bf92f3577b34da6a3ce929d0e0e4736",
				SpanID:        "00f067aa0ba902b7",
				ServiceName:   "checkout-api",
				OperationName: "POST /checkout",
				StartTime:     now.Add(10 * time.Minute),
			},
		}
		if err := ValidateSpans(spans); err == nil {
			t.Errorf("expected error when start_time is in future")
		}
	})
}

func TestEnrichSpans(t *testing.T) {
	now := time.Now()
	rawSpans := []domain.Span{
		{
			TraceID:       "trace-1",
			SpanID:        "span-1",
			ServiceName:   "  Checkout-API  ", // Leading/trailing whitespace
			OperationName: "Process_Checkout",
			StartTime:     now,
			OrgID:         "spoofed-org-999", // Attempted spoof
		},
	}

	enriched := EnrichSpans(rawSpans, "authentic-org-111")
	if len(enriched) != 1 {
		t.Fatalf("expected 1 enriched span, got %d", len(enriched))
	}

	s := enriched[0]
	if s.OrgID != "authentic-org-111" {
		t.Errorf("expected OrgID to be overwritten by authentic org, got %s", s.OrgID)
	}

	if s.ServiceName != "checkout-api" {
		t.Errorf("expected normalized lower-case trimmed service_name, got %s", s.ServiceName)
	}

	if s.ReceivedAt.IsZero() {
		t.Errorf("expected ReceivedAt timestamp to be populated")
	}

	// Verify nil tags map initialized
	if s.Tags == nil {
		t.Errorf("expected Tags map to be initialized")
	}
}

package ingest

import (
	"fmt"
	"strings"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
)

const (
	// MaxBatchSpans limits the maximum number of spans accepted per batch (10,000).
	// Accommodates default OpenTelemetry Collector batchprocessor configuration (send_batch_size: 8192).
	MaxBatchSpans = 10000
)

type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed: %s", strings.Join(e.Errors, "; "))
}

func ValidateSpans(spans []domain.Span) error {
	if len(spans) == 0 {
		return &ValidationError{Errors: []string{"empty span batch"}}
	}
	if len(spans) > MaxBatchSpans {
		return &ValidationError{Errors: []string{fmt.Sprintf("batch size exceeds maximum limit of %d spans", MaxBatchSpans)}}
	}

	var errs []string

	for i, s := range spans {
		var spanErrs []string
		if s.TraceID == "" {
			spanErrs = append(spanErrs, "empty trace_id")
		}
		if s.SpanID == "" {
			spanErrs = append(spanErrs, "empty span_id")
		}
		if s.ServiceName == "" {
			spanErrs = append(spanErrs, "empty service_name")
		}
		if s.StartTime.IsZero() {
			spanErrs = append(spanErrs, "invalid start_time")
		}
		if s.DurationMs == 0 {
			spanErrs = append(spanErrs, "duration_ms must be greater than zero")
		}

		if len(spanErrs) > 0 {
			errs = append(errs, fmt.Sprintf("span[%d]: %s", i, strings.Join(spanErrs, ", ")))
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}

	return nil
}

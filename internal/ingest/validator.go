package ingest

import (
	"fmt"
	"strings"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
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
	if len(spans) > 1000 {
		return &ValidationError{Errors: []string{"batch size exceeds maximum limit of 1000 spans"}}
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

package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
)

type TraceRepository struct {
	conn clickhouse.Conn
}

func NewTraceRepository(conn clickhouse.Conn) *TraceRepository {
	return &TraceRepository{conn: conn}
}

func (r *TraceRepository) InsertSpansBatch(ctx context.Context, spans []domain.Span) error {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "InsertSpansBatch")
	defer timer.ObserveDuration()

	batch, err := r.conn.PrepareBatch(ctx, "INSERT INTO otel_spans")
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "PrepareBatch").Inc()
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	for _, s := range spans {
		err = batch.Append(
			s.OrgID,
			s.TraceID,
			s.SpanID,
			s.ParentSpanID,
			s.ServiceName,
			s.OperationName,
			s.DurationMs,
			s.StatusCode,
			s.ErrorMessage,
			s.Tags,
			s.StartTime,
		)
		if err != nil {
			observability.DbErrorsTotal.WithLabelValues("clickhouse", "AppendBatch").Inc()
			return fmt.Errorf("failed to append to batch: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "SendBatch").Inc()
		return fmt.Errorf("failed to send batch: %w", err)
	}

	return nil
}

func (r *TraceRepository) SearchTraces(ctx context.Context, orgID string, filters domain.TraceFilter) ([]domain.TraceRow, error) {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "SearchTraces")
	defer timer.ObserveDuration()

	var queryParts []string
	var args []interface{}

	// Query comments for Request ID propagation
	reqID := observability.GetRequestID(ctx)
	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	// Filter by Org ID for strict tenant isolation
	queryParts = append(queryParts, "org_id = ?")
	args = append(args, orgID)

	if filters.Service != "" && filters.Service != "all" {
		queryParts = append(queryParts, "service_name = ?")
		args = append(args, filters.Service)
	}

	if !filters.From.IsZero() {
		queryParts = append(queryParts, "start_time >= ?")
		args = append(args, filters.From)
	}

	if !filters.To.IsZero() {
		queryParts = append(queryParts, "start_time <= ?")
		args = append(args, filters.To)
	}

	if filters.MinDurationMs > 0 {
		queryParts = append(queryParts, "duration_ms >= ?")
		args = append(args, uint32(filters.MinDurationMs))
	}

	if filters.MaxDurationMs > 0 {
		queryParts = append(queryParts, "duration_ms <= ?")
		args = append(args, uint32(filters.MaxDurationMs))
	}

	if filters.Status == "error" {
		queryParts = append(queryParts, "status_code = 2")
	} else if filters.Status == "success" {
		queryParts = append(queryParts, "status_code = 1")
	}

	whereClause := strings.Join(queryParts, " AND ")

	sql := fmt.Sprintf(`
		SELECT 
			trace_id,
			any(service_name) as root_service,
			any(operation_name) as root_operation,
			max(duration_ms) as duration_ms,
			count() as span_count,
			sum(status_code = 2) as error_count,
			min(start_time) as start_time
		FROM otel_spans
		WHERE %s
		GROUP BY trace_id
		ORDER BY start_time DESC
		LIMIT ? OFFSET ? %s`, whereClause, comment)

	limit := filters.Limit
	if limit == 0 {
		limit = 50
	}
	args = append(args, limit, filters.Offset)

	rows, err := r.conn.Query(ctx, sql, args...)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "SearchTraces").Inc()
		return nil, fmt.Errorf("failed to search traces: %w", err)
	}
	defer rows.Close()

	var traces []domain.TraceRow
	for rows.Next() {
		var tr domain.TraceRow
		var startTime time.Time
		if err := rows.Scan(
			&tr.TraceID,
			&tr.RootService,
			&tr.RootOperation,
			&tr.DurationMs,
			&tr.SpanCount,
			&tr.ErrorCount,
			&startTime,
		); err != nil {
			return nil, fmt.Errorf("failed to scan trace row: %w", err)
		}
		tr.StartTime = startTime.UnixNano() / int64(time.Millisecond)
		traces = append(traces, tr)
	}

	return traces, nil
}

func (r *TraceRepository) GetTraceWithSpans(ctx context.Context, orgID string, traceID string) (*domain.TraceDetail, error) {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "GetTraceWithSpans")
	defer timer.ObserveDuration()

	reqID := observability.GetRequestID(ctx)
	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	sql := fmt.Sprintf(`
		SELECT 
			trace_id,
			span_id,
			parent_span_id,
			service_name,
			operation_name,
			duration_ms,
			status_code,
			error_message,
			tags,
			start_time
		FROM otel_spans
		WHERE org_id = ? AND trace_id = ? %s`, comment)

	rows, err := r.conn.Query(ctx, sql, orgID, traceID)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "GetTraceWithSpans").Inc()
		return nil, fmt.Errorf("failed to get trace with spans: %w", err)
	}
	defer rows.Close()

	var spans []domain.Span
	var minStartTime time.Time
	var maxDuration uint32
	var rootService, rootOperation string

	for rows.Next() {
		var s domain.Span
		var startTime time.Time
		if err := rows.Scan(
			&s.TraceID,
			&s.SpanID,
			&s.ParentSpanID,
			&s.ServiceName,
			&s.OperationName,
			&s.DurationMs,
			&s.StatusCode,
			&s.ErrorMessage,
			&s.Tags,
			&startTime,
		); err != nil {
			return nil, fmt.Errorf("failed to scan span: %w", err)
		}
		s.StartTime = startTime
		if minStartTime.IsZero() || startTime.Before(minStartTime) {
			minStartTime = startTime
			rootService = s.ServiceName
			rootOperation = s.OperationName
		}
		if s.DurationMs > maxDuration {
			maxDuration = s.DurationMs
		}
		spans = append(spans, s)
	}

	if len(spans) == 0 {
		return nil, nil
	}

	errorCount := 0
	for i := range spans {
		spans[i].StartOffsetMs = uint32(spans[i].StartTime.Sub(minStartTime).Milliseconds())
		if spans[i].StatusCode == 2 {
			errorCount++
		}
	}

	return &domain.TraceDetail{
		TraceID:       traceID,
		RootService:   rootService,
		RootOperation: rootOperation,
		DurationMs:    maxDuration, // Simplified approximation of trace duration
		SpanCount:     len(spans),
		ErrorCount:    errorCount,
		StartTime:     minStartTime.UnixNano() / int64(time.Millisecond),
		Spans:         spans,
	}, nil
}

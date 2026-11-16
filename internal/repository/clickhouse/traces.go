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

// BuildSearchTracesQuery constructs the production SQL query and arguments for SearchTraces.
func (r *TraceRepository) BuildSearchTracesQuery(orgID string, filters domain.TraceFilter, reqID string) (string, []interface{}) {
	var queryParts []string
	var args []interface{}

	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	// Filter by Org ID for strict tenant isolation
	queryParts = append(queryParts, "otel_spans.org_id = ?")
	args = append(args, orgID)

	if filters.Service != "" && filters.Service != "all" {
		queryParts = append(queryParts, "otel_spans.service_name = ?")
		args = append(args, filters.Service)
	}

	if !filters.From.IsZero() {
		queryParts = append(queryParts, "otel_spans.start_time >= ?")
		args = append(args, filters.From)
	}

	if !filters.To.IsZero() {
		queryParts = append(queryParts, "otel_spans.start_time <= ?")
		args = append(args, filters.To)
	}

	if filters.MinDurationMs > 0 {
		queryParts = append(queryParts, "otel_spans.duration_ms >= ?")
		args = append(args, uint32(filters.MinDurationMs))
	}

	if filters.MaxDurationMs > 0 {
		queryParts = append(queryParts, "otel_spans.duration_ms <= ?")
		args = append(args, uint32(filters.MaxDurationMs))
	}

	if filters.Status == "error" {
		queryParts = append(queryParts, "otel_spans.status_code = ?")
		args = append(args, uint8(2))
	} else if filters.Status == "success" {
		queryParts = append(queryParts, "otel_spans.status_code = ?")
		args = append(args, uint8(1))
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
			min(start_time) as min_start_time
		FROM otel_spans
		WHERE %s
		GROUP BY trace_id
		ORDER BY min_start_time DESC
		LIMIT ? OFFSET ? %s`, whereClause, comment)

	limit := filters.Limit
	if limit == 0 {
		limit = 50
	}
	args = append(args, limit, filters.Offset)

	return sql, args
}

// ExplainSearchTraces derives the EXPLAIN plan directly from the production query path.
func (r *TraceRepository) ExplainSearchTraces(ctx context.Context, orgID string, filters domain.TraceFilter) (string, error) {
	sql, args := r.BuildSearchTracesQuery(orgID, filters, "")
	return explainQuery(ctx, r.conn, sql, args)
}

func (r *TraceRepository) SearchTraces(ctx context.Context, orgID string, filters domain.TraceFilter) ([]domain.TraceRow, error) {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "SearchTraces")
	defer timer.ObserveDuration()

	reqID := observability.GetRequestID(ctx)
	sql, args := r.BuildSearchTracesQuery(orgID, filters, reqID)

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
		var spanCount uint64
		var errorCount uint64
		if err := rows.Scan(
			&tr.TraceID,
			&tr.RootService,
			&tr.RootOperation,
			&tr.DurationMs,
			&spanCount,
			&errorCount,
			&startTime,
		); err != nil {
			return nil, fmt.Errorf("failed to scan trace row: %w", err)
		}
		tr.SpanCount = int(spanCount)
		tr.ErrorCount = int(errorCount)
		tr.StartTime = startTime.UnixNano() / int64(time.Millisecond)
		traces = append(traces, tr)
	}

	return traces, nil
}

// BuildGetTraceWithSpansQuery constructs the production SQL query and arguments for GetTraceWithSpans.
func (r *TraceRepository) BuildGetTraceWithSpansQuery(orgID, traceID, reqID string) (string, []interface{}) {
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

	return sql, []interface{}{orgID, traceID}
}

// ExplainGetTraceWithSpans derives the EXPLAIN plan directly from the production query path.
func (r *TraceRepository) ExplainGetTraceWithSpans(ctx context.Context, orgID, traceID string) (string, error) {
	sql, args := r.BuildGetTraceWithSpansQuery(orgID, traceID, "")
	return explainQuery(ctx, r.conn, sql, args)
}

func (r *TraceRepository) GetTraceWithSpans(ctx context.Context, orgID string, traceID string) (*domain.TraceDetail, error) {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "GetTraceWithSpans")
	defer timer.ObserveDuration()

	reqID := observability.GetRequestID(ctx)
	sql, args := r.BuildGetTraceWithSpansQuery(orgID, traceID, reqID)

	rows, err := r.conn.Query(ctx, sql, args...)
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

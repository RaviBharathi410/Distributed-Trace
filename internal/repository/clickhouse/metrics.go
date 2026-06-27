package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
)

type MetricsRepository struct {
	conn clickhouse.Conn
}

func NewMetricsRepository(conn clickhouse.Conn) *MetricsRepository {
	return &MetricsRepository{conn: conn}
}

func (r *MetricsRepository) GetLatencyTimeSeries(ctx context.Context, orgID, service string, from, to time.Time, percent string) ([]domain.MetricPoint, error) {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "GetLatencyTimeSeries")
	defer timer.ObserveDuration()

	reqID := observability.GetRequestID(ctx)
	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	percentileCol := "quantile(0.99)(duration_ms)"
	if percent == "p95" {
		percentileCol = "quantile(0.95)(duration_ms)"
	} else if percent == "p50" {
		percentileCol = "quantile(0.50)(duration_ms)"
	}

	// Query raw spans for real-time granularity
	sql := fmt.Sprintf(`
		SELECT 
			toStartOfInterval(start_time, INTERVAL 1 MINUTE) as ts,
			%s as val
		FROM otel_spans
		WHERE service_name = ? AND start_time >= ? AND start_time <= ?
		GROUP BY ts
		ORDER BY ts ASC %s`, percentileCol, comment)

	rows, err := r.conn.Query(ctx, sql, service, from, to)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "GetLatencyTimeSeries").Inc()
		return nil, fmt.Errorf("failed to query latency time series: %w", err)
	}
	defer rows.Close()

	var points []domain.MetricPoint
	for rows.Next() {
		var p domain.MetricPoint
		if err := rows.Scan(&p.Timestamp, &p.Value); err != nil {
			return nil, fmt.Errorf("failed to scan metric point: %w", err)
		}
		points = append(points, p)
	}

	return points, nil
}

func (r *MetricsRepository) GetThroughputTimeSeries(ctx context.Context, orgID, service string, from, to time.Time) ([]domain.MetricPoint, error) {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "GetThroughputTimeSeries")
	defer timer.ObserveDuration()

	reqID := observability.GetRequestID(ctx)
	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	// Query count of spans divided by 60s for RPS
	sql := fmt.Sprintf(`
		SELECT 
			toStartOfInterval(start_time, INTERVAL 1 MINUTE) as ts,
			count() / 60.0 as val
		FROM otel_spans
		WHERE service_name = ? AND start_time >= ? AND start_time <= ?
		GROUP BY ts
		ORDER BY ts ASC %s`, comment)

	rows, err := r.conn.Query(ctx, sql, service, from, to)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "GetThroughputTimeSeries").Inc()
		return nil, fmt.Errorf("failed to query throughput time series: %w", err)
	}
	defer rows.Close()

	var points []domain.MetricPoint
	for rows.Next() {
		var p domain.MetricPoint
		if err := rows.Scan(&p.Timestamp, &p.Value); err != nil {
			return nil, fmt.Errorf("failed to scan metric point: %w", err)
		}
		points = append(points, p)
	}

	return points, nil
}

func (r *MetricsRepository) GetErrorsTimeSeries(ctx context.Context, orgID, service string, from, to time.Time) ([]domain.MetricPoint, error) {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "GetErrorsTimeSeries")
	defer timer.ObserveDuration()

	reqID := observability.GetRequestID(ctx)
	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	// Query error rate
	sql := fmt.Sprintf(`
		SELECT 
			toStartOfInterval(start_time, INTERVAL 1 MINUTE) as ts,
			sum(status_code = 2) * 100.0 / count() as val
		FROM otel_spans
		WHERE service_name = ? AND start_time >= ? AND start_time <= ?
		GROUP BY ts
		ORDER BY ts ASC %s`, comment)

	rows, err := r.conn.Query(ctx, sql, service, from, to)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "GetErrorsTimeSeries").Inc()
		return nil, fmt.Errorf("failed to query errors time series: %w", err)
	}
	defer rows.Close()

	var points []domain.MetricPoint
	for rows.Next() {
		var p domain.MetricPoint
		if err := rows.Scan(&p.Timestamp, &p.Value); err != nil {
			return nil, fmt.Errorf("failed to scan metric point: %w", err)
		}
		points = append(points, p)
	}

	return points, nil
}

package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
)

type ServiceRepository struct {
	conn clickhouse.Conn
}

func NewServiceRepository(conn clickhouse.Conn) *ServiceRepository {
	return &ServiceRepository{conn: conn}
}

func (r *ServiceRepository) ListServices(ctx context.Context) ([]string, error) {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "ListServices")
	defer timer.ObserveDuration()

	reqID := observability.GetRequestID(ctx)
	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	sql := fmt.Sprintf(`SELECT DISTINCT service_name FROM otel_spans %s`, comment)

	rows, err := r.conn.Query(ctx, sql)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "ListServices").Inc()
		return nil, fmt.Errorf("failed to query service names: %w", err)
	}
	defer rows.Close()

	var services []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("failed to scan service name: %w", err)
		}
		services = append(services, s)
	}
	return services, nil
}

func (r *ServiceRepository) GetServiceStats(ctx context.Context, serviceName string, from, to time.Time) (*domain.ServiceStats, error) {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "GetServiceStats")
	defer timer.ObserveDuration()

	reqID := observability.GetRequestID(ctx)
	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	sql := fmt.Sprintf(`
		SELECT 
			quantile(0.50)(duration_ms) as p50,
			quantile(0.95)(duration_ms) as p95,
			quantile(0.99)(duration_ms) as p99,
			sum(status_code = 2) * 100.0 / count() as error_rate,
			count() / (toUnixTimestamp(?) - toUnixTimestamp(?)) as request_rate
		FROM otel_spans
		WHERE service_name = ? AND start_time >= ? AND start_time <= ?
		%s`, comment)

	var stats domain.ServiceStats
	stats.ServiceName = serviceName
	stats.TimeRangeFrom = from
	stats.TimeRangeTo = to

	err := r.conn.QueryRow(ctx, sql, to, from, serviceName, from, to).Scan(
		&stats.P50Ms,
		&stats.P95Ms,
		&stats.P99Ms,
		&stats.ErrorRatePct,
		&stats.RequestRate,
	)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "GetServiceStats").Inc()
		return nil, fmt.Errorf("failed to compute stats for service: %w", err)
	}

	return &stats, nil
}

func (r *ServiceRepository) GetServiceGraph(ctx context.Context, from, to time.Time) (*domain.ServiceGraph, error) {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "GetServiceGraph")
	defer timer.ObserveDuration()

	reqID := observability.GetRequestID(ctx)
	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	// 1. Build edges by matching parent_span_id with span_id of other services
	edgesSQL := fmt.Sprintf(`
		SELECT 
			p.service_name as source,
			c.service_name as target,
			count() / (toUnixTimestamp(?) - toUnixTimestamp(?)) as rps,
			sum(c.status_code = 2) * 100.0 / count() as error_rate
		FROM otel_spans c
		JOIN otel_spans p ON c.parent_span_id = p.span_id AND c.trace_id = p.trace_id
		WHERE c.start_time >= ? AND c.start_time <= ? AND c.service_name != p.service_name
		GROUP BY source, target
		%s`, comment)

	rows, err := r.conn.Query(ctx, edgesSQL, to, from, from, to)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "GetServiceGraphEdges").Inc()
		return nil, fmt.Errorf("failed to query service graph edges: %w", err)
	}
	defer rows.Close()

	var edges []domain.ServiceEdge
	for rows.Next() {
		var e domain.ServiceEdge
		if err := rows.Scan(&e.Source, &e.Target, &e.RPS, &e.ErrorRatePct); err != nil {
			return nil, fmt.Errorf("failed to scan edge: %w", err)
		}
		edges = append(edges, e)
	}

	// 2. Fetch service nodes with p99 and status info
	nodesSQL := fmt.Sprintf(`
		SELECT 
			service_name,
			quantile(0.99)(duration_ms) as p99,
			sum(status_code = 2) * 100.0 / count() as error_rate
		FROM otel_spans
		WHERE start_time >= ? AND start_time <= ?
		GROUP BY service_name
		%s`, comment)

	nRows, err := r.conn.Query(ctx, nodesSQL, from, to)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "GetServiceGraphNodes").Inc()
		return nil, fmt.Errorf("failed to query service graph nodes: %w", err)
	}
	defer nRows.Close()

	var nodes []domain.ServiceNode
	for nRows.Next() {
		var n domain.ServiceNode
		var errRate float64
		if err := nRows.Scan(&n.ID, &n.P99Ms, &errRate); err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}
		n.Name = n.ID
		if errRate > 5.0 {
			n.Health = "critical"
		} else if errRate > 1.0 {
			n.Health = "degraded"
		} else {
			n.Health = "healthy"
		}
		nodes = append(nodes, n)
	}

	// Simple critical path identification (find path with max RPS or max latency chain)
	// For demo/simplicity, we mark edges with errorRate > 5% as critical or highest RPS
	if len(edges) > 0 {
		maxRpsIdx := 0
		for i := 1; i < len(edges); i++ {
			if edges[i].RPS > edges[maxRpsIdx].RPS {
				maxRpsIdx = i
			}
		}
		edges[maxRpsIdx].CriticalPath = true
	}

	return &domain.ServiceGraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

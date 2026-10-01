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

func (r *ServiceRepository) ListServices(ctx context.Context, orgID string) ([]string, error) {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "ListServices")
	defer timer.ObserveDuration()

	reqID := observability.GetRequestID(ctx)
	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	sql := fmt.Sprintf(`SELECT DISTINCT service_name FROM otel_spans WHERE org_id = ? %s`, comment)

	rows, err := r.conn.Query(ctx, sql, orgID)
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

func (r *ServiceRepository) GetServiceStats(ctx context.Context, orgID string, serviceName string, from, to time.Time) (*domain.ServiceStats, error) {
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
		WHERE org_id = ? AND service_name = ? AND start_time >= ? AND start_time <= ?
		%s`, comment)

	var stats domain.ServiceStats
	stats.ServiceName = serviceName
	stats.TimeRangeFrom = from
	stats.TimeRangeTo = to

	err := r.conn.QueryRow(ctx, sql, to, from, orgID, serviceName, from, to).Scan(
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

func (r *ServiceRepository) GetServiceGraph(ctx context.Context, orgID string, from, to time.Time) (*domain.ServiceGraph, error) {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "GetServiceGraph")
	defer timer.ObserveDuration()

	reqID := observability.GetRequestID(ctx)
	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	// 1. Build edges by matching parent_span_id with span_id of other services within the same org
	edgesSQL := fmt.Sprintf(`
		SELECT 
			p.service_name as source,
			c.service_name as target,
			count() / (toUnixTimestamp(?) - toUnixTimestamp(?)) as rps,
			sum(c.status_code = 2) * 100.0 / count() as error_rate,
			quantile(0.95)(c.duration_ms) as p95_ms
		FROM otel_spans c
		JOIN otel_spans p ON c.parent_span_id = p.span_id AND c.trace_id = p.trace_id AND c.org_id = p.org_id
		WHERE c.org_id = ? AND c.start_time >= ? AND c.start_time <= ? AND c.service_name != p.service_name
		GROUP BY source, target
		%s`, comment)

	rows, err := r.conn.Query(ctx, edgesSQL, to, from, orgID, from, to)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "GetServiceGraphEdges").Inc()
		return nil, fmt.Errorf("failed to query service graph edges: %w", err)
	}
	defer rows.Close()

	var edges []domain.ServiceEdge
	for rows.Next() {
		var e domain.ServiceEdge
		if err := rows.Scan(&e.Source, &e.Target, &e.RPS, &e.ErrorRatePct, &e.P95Ms); err != nil {
			return nil, fmt.Errorf("failed to scan edge: %w", err)
		}
		edges = append(edges, e)
	}

	// 2. Fetch active unresolved anomalies per service within the same org (Phase 3 single source of truth)
	anomaliesSQL := fmt.Sprintf(`
		SELECT 
			service_name,
			countIf(severity = 'critical') as crit_count,
			countIf(severity = 'high') as high_count,
			countIf(severity = 'medium') as med_count,
			count() as total_active
		FROM anomalies
		WHERE org_id = ? AND status != 'resolved'
		GROUP BY service_name
		%s`, comment)

	aRows, err := r.conn.Query(ctx, anomaliesSQL, orgID)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "GetServiceGraphAnomalies").Inc()
		return nil, fmt.Errorf("failed to query active anomalies for service graph: %w", err)
	}
	defer aRows.Close()

	type serviceAnomalyStats struct {
		critical uint64
		high     uint64
		medium   uint64
		total    uint64
	}
	activeAnomalies := make(map[string]serviceAnomalyStats)
	for aRows.Next() {
		var sName string
		var crit, high, med, total uint64
		if err := aRows.Scan(&sName, &crit, &high, &med, &total); err != nil {
			return nil, fmt.Errorf("failed to scan service anomalies row: %w", err)
		}
		activeAnomalies[sName] = serviceAnomalyStats{
			critical: crit,
			high:     high,
			medium:   med,
			total:    total,
		}
	}

	// 3. Fetch service nodes with p99 and error rates within the same org
	nodesSQL := fmt.Sprintf(`
		SELECT 
			service_name,
			quantile(0.99)(duration_ms) as p99,
			sum(status_code = 2) * 100.0 / count() as error_rate
		FROM otel_spans
		WHERE org_id = ? AND start_time >= ? AND start_time <= ?
		GROUP BY service_name
		%s`, comment)

	nRows, err := r.conn.Query(ctx, nodesSQL, orgID, from, to)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "GetServiceGraphNodes").Inc()
		return nil, fmt.Errorf("failed to query service graph nodes: %w", err)
	}
	defer nRows.Close()

	var nodes []domain.ServiceNode
	seenNodes := make(map[string]bool)
	for nRows.Next() {
		var n domain.ServiceNode
		var errRate float64
		if err := nRows.Scan(&n.ID, &n.P99Ms, &errRate); err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}
		n.Name = n.ID
		seenNodes[n.ID] = true

		anom := activeAnomalies[n.ID]
		n.ActiveAnomalies = int(anom.total)

		// Unified health evaluation driven by Phase 3 anomalies + error rates
		if anom.critical > 0 || anom.high > 0 || errRate > 5.0 {
			n.Health = "critical"
		} else if anom.medium > 0 || errRate > 1.0 {
			n.Health = "degraded"
		} else {
			n.Health = "healthy"
		}
		nodes = append(nodes, n)
	}

	// Ensure any service with an active anomaly is visible on the graph even if idle in from..to
	for sName, anom := range activeAnomalies {
		if !seenNodes[sName] {
			health := "healthy"
			if anom.critical > 0 || anom.high > 0 {
				health = "critical"
			} else if anom.medium > 0 {
				health = "degraded"
			}
			nodes = append(nodes, domain.ServiceNode{
				ID:              sName,
				Name:            sName,
				Health:          health,
				ActiveAnomalies: int(anom.total),
			})
			seenNodes[sName] = true
		}
	}

	// Critical path identification: prioritize edges flowing into critical/degraded bottleneck nodes
	if len(edges) > 0 {
		critServiceMap := make(map[string]bool)
		for _, n := range nodes {
			if n.Health == "critical" {
				critServiceMap[n.ID] = true
			}
		}

		marked := false
		for i := range edges {
			if critServiceMap[edges[i].Target] {
				edges[i].CriticalPath = true
				marked = true
			}
		}

		if !marked {
			maxRpsIdx := 0
			for i := 1; i < len(edges); i++ {
				if edges[i].RPS > edges[maxRpsIdx].RPS {
					maxRpsIdx = i
				}
			}
			edges[maxRpsIdx].CriticalPath = true
		}
	}

	return &domain.ServiceGraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

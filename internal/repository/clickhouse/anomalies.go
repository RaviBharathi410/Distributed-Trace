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

type AnomalyRepository struct {
	conn clickhouse.Conn
}

func NewAnomalyRepository(conn clickhouse.Conn) *AnomalyRepository {
	return &AnomalyRepository{conn: conn}
}

func (r *AnomalyRepository) Insert(ctx context.Context, a *domain.Anomaly) error {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "InsertAnomaly")
	defer timer.ObserveDuration()

	reqID := observability.GetRequestID(ctx)
	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	q := fmt.Sprintf(`
		INSERT INTO anomalies (
			id, org_id, service_name, operation_name, root_cause_service,
			severity, z_score, baseline_latency_ms, observed_latency_ms,
			error_rate_delta, status, detected_at, resolved_at, root_cause_path
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) %s`, comment)

	err := r.conn.Exec(ctx, q,
		a.ID, a.OrgID, a.ServiceName, a.OperationName, a.RootCauseService,
		string(a.Severity), a.ZScore, a.BaselineLatencyMs, a.ObservedLatencyMs,
		a.ErrorRateDelta, a.Status, a.DetectedAt, a.ResolvedAt, a.RootCausePath,
	)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "InsertAnomaly").Inc()
		return fmt.Errorf("failed to insert anomaly: %w", err)
	}

	observability.AnomaliesDetected.WithLabelValues(string(a.Severity)).Inc()
	return nil
}

func (r *AnomalyRepository) List(ctx context.Context, orgID string, filters domain.AnomalyFilter) ([]domain.Anomaly, error) {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "ListAnomalies")
	defer timer.ObserveDuration()

	var queryParts []string
	var args []interface{}

	reqID := observability.GetRequestID(ctx)
	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	queryParts = append(queryParts, "org_id = ?")
	args = append(args, orgID)

	if filters.Severity != "" && filters.Severity != "all" {
		queryParts = append(queryParts, "severity = ?")
		args = append(args, filters.Severity)
	}

	if filters.Service != "" && filters.Service != "all" {
		queryParts = append(queryParts, "service_name = ?")
		args = append(args, filters.Service)
	}

	if filters.Status != "" && filters.Status != "all" {
		queryParts = append(queryParts, "status = ?")
		args = append(args, filters.Status)
	}

	whereClause := strings.Join(queryParts, " AND ")

	sql := fmt.Sprintf(`
		SELECT 
			id, org_id, service_name, operation_name, root_cause_service,
			severity, z_score, baseline_latency_ms, observed_latency_ms,
			error_rate_delta, status, detected_at, resolved_at, root_cause_path
		FROM anomalies
		WHERE %s
		ORDER BY detected_at DESC
		LIMIT ? OFFSET ? %s`, whereClause, comment)

	limit := filters.Limit
	if limit == 0 {
		limit = 50
	}
	args = append(args, limit, filters.Offset)

	rows, err := r.conn.Query(ctx, sql, args...)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "ListAnomalies").Inc()
		return nil, fmt.Errorf("failed to query anomalies: %w", err)
	}
	defer rows.Close()

	var anomalies []domain.Anomaly
	for rows.Next() {
		var a domain.Anomaly
		var severityStr string
		var detectedAt time.Time
		var resolvedAt *time.Time
		if err := rows.Scan(
			&a.ID, &a.OrgID, &a.ServiceName, &a.OperationName, &a.RootCauseService,
			&severityStr, &a.ZScore, &a.BaselineLatencyMs, &a.ObservedLatencyMs,
			&a.ErrorRateDelta, &a.Status, &detectedAt, &resolvedAt, &a.RootCausePath,
		); err != nil {
			return nil, fmt.Errorf("failed to scan anomaly: %w", err)
		}
		a.Severity = domain.Severity(severityStr)
		a.DetectedAt = detectedAt
		a.ResolvedAt = resolvedAt
		anomalies = append(anomalies, a)
	}

	return anomalies, nil
}

func (r *AnomalyRepository) GetByID(ctx context.Context, orgID string, id string) (*domain.Anomaly, error) {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "GetAnomalyByID")
	defer timer.ObserveDuration()

	reqID := observability.GetRequestID(ctx)
	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	sql := fmt.Sprintf(`
		SELECT 
			id, org_id, service_name, operation_name, root_cause_service,
			severity, z_score, baseline_latency_ms, observed_latency_ms,
			error_rate_delta, status, detected_at, resolved_at, root_cause_path
		FROM anomalies
		WHERE id = ? AND org_id = ? 
		LIMIT 1 %s`, comment)

	rows, err := r.conn.Query(ctx, sql, id, orgID)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "GetAnomalyByID").Inc()
		return nil, fmt.Errorf("failed to query anomaly: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	var a domain.Anomaly
	var severityStr string
	var detectedAt time.Time
	var resolvedAt *time.Time
	if err := rows.Scan(
		&a.ID, &a.OrgID, &a.ServiceName, &a.OperationName, &a.RootCauseService,
		&severityStr, &a.ZScore, &a.BaselineLatencyMs, &a.ObservedLatencyMs,
		&a.ErrorRateDelta, &a.Status, &detectedAt, &resolvedAt, &a.RootCausePath,
	); err != nil {
		return nil, fmt.Errorf("failed to scan anomaly detail: %w", err)
	}
	a.Severity = domain.Severity(severityStr)
	a.DetectedAt = detectedAt
	a.ResolvedAt = resolvedAt

	return &a, nil
}

func (r *AnomalyRepository) UpdateStatus(ctx context.Context, orgID string, id string, status string) error {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "UpdateAnomalyStatus")
	defer timer.ObserveDuration()

	reqID := observability.GetRequestID(ctx)
	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	var sql string
	if status == "resolved" {
		sql = fmt.Sprintf(`ALTER TABLE anomalies UPDATE status = ?, resolved_at = ? WHERE id = ? AND org_id = ? %s`, comment)
		err := r.conn.Exec(ctx, sql, status, time.Now(), id, orgID)
		if err != nil {
			observability.DbErrorsTotal.WithLabelValues("clickhouse", "UpdateAnomalyStatus").Inc()
			return fmt.Errorf("failed to update anomaly: %w", err)
		}
	} else {
		sql = fmt.Sprintf(`ALTER TABLE anomalies UPDATE status = ? WHERE id = ? AND org_id = ? %s`, comment)
		err := r.conn.Exec(ctx, sql, status, id, orgID)
		if err != nil {
			observability.DbErrorsTotal.WithLabelValues("clickhouse", "UpdateAnomalyStatus").Inc()
			return fmt.Errorf("failed to update anomaly: %w", err)
		}
	}

	return nil
}

func (r *AnomalyRepository) GetStats(ctx context.Context, orgID string) (*domain.AnomalyStats, error) {
	timer := observability.DbQueryDuration.WithLabelValues("clickhouse", "GetAnomalyStats")
	defer timer.ObserveDuration()

	reqID := observability.GetRequestID(ctx)
	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	sql := fmt.Sprintf(`
		SELECT 
			sum(severity = 'critical') as critical,
			sum(severity = 'high') as high,
			sum(severity = 'medium') as medium,
			sum(severity = 'low') as low
		FROM anomalies
		WHERE org_id = ? AND status != 'resolved' %s`, comment)

	var stats domain.AnomalyStats
	var critical, high, medium, low uint64
	err := r.conn.QueryRow(ctx, sql, orgID).Scan(&critical, &high, &medium, &low)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("clickhouse", "GetAnomalyStats").Inc()
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	stats.Critical = int(critical)
	stats.High = int(high)
	stats.Medium = int(medium)
	stats.Low = int(low)

	return &stats, nil
}

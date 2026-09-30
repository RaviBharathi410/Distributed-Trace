package analysis

import (
	"context"
	"fmt"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
)

type AnomalyWriter interface {
	Insert(ctx context.Context, a *domain.Anomaly) error
}

type AnomalyDetector struct {
	baseline *BaselineCalculator
	causal   *CausalAnalyzer
	writer   AnomalyWriter
}

func NewAnomalyDetector(baseline *BaselineCalculator, causal *CausalAnalyzer, writer AnomalyWriter) *AnomalyDetector {
	if baseline == nil {
		baseline = NewBaselineCalculator()
	}
	if causal == nil {
		causal = NewCausalAnalyzer()
	}
	return &AnomalyDetector{
		baseline: baseline,
		causal:   causal,
		writer:   writer,
	}
}

// CalculateSeverity maps standard deviation distance (Z-score) to discrete alert severity tiers.
func CalculateSeverity(zScore float64) domain.Severity {
	switch {
	case zScore >= 5.0:
		return domain.SeverityCritical
	case zScore >= 4.0:
		return domain.SeverityHigh
	case zScore >= 3.0:
		return domain.SeverityMedium
	default:
		return domain.SeverityLow
	}
}

// AnalyzeTrace inspects a set of spans belonging to a single trace, assesses root span deviation,
// traverses causal dependencies to locate downstream root causes, and persists detected anomalies.
func (d *AnomalyDetector) AnalyzeTrace(ctx context.Context, orgID string, spans []domain.Span) (*domain.Anomaly, error) {
	if len(spans) == 0 {
		return nil, nil
	}

	// 1. Identify root span
	var root *domain.Span
	for i := range spans {
		if spans[i].ParentSpanID == "" {
			root = &spans[i]
			break
		}
	}
	if root == nil {
		// Fallback: earliest span
		root = &spans[0]
		for i := 1; i < len(spans); i++ {
			if spans[i].StartTime.Before(root.StartTime) {
				root = &spans[i]
			}
		}
	}

	// 2. Statistical baseline evaluation
	zScore, baselineMean, isAnomalous := d.baseline.CalculateZScore(
		orgID,
		root.ServiceName,
		root.OperationName,
		float64(root.DurationMs),
	)

	if !isAnomalous {
		return nil, nil
	}

	// 3. Causal graph traversal to find root cause
	causalRes := d.causal.AnalyzeRootCause(spans)

	// 4. Compute error rate delta
	var errorCount int
	for _, s := range spans {
		if s.StatusCode == 2 {
			errorCount++
		}
	}
	var errorRateDelta float64
	if len(spans) > 0 {
		errorRateDelta = float64(errorCount) / float64(len(spans)) * 100.0
	}

	severity := CalculateSeverity(zScore)
	anomalyID := fmt.Sprintf("anom-%s-%d", root.TraceID, time.Now().UnixNano()%1000000)

	anomaly := &domain.Anomaly{
		ID:                 anomalyID,
		OrgID:              orgID,
		ServiceName:        root.ServiceName,
		OperationName:      root.OperationName,
		RootCauseService:   causalRes.RootCauseService,
		Severity:           severity,
		ZScore:             zScore,
		BaselineLatencyMs:  baselineMean,
		ObservedLatencyMs:  float64(root.DurationMs),
		ErrorRateDelta:     errorRateDelta,
		Status:             "open",
		DetectedAt:         root.StartTime,
		RootCausePath:      causalRes.RootCausePath,
	}

	// 5. Persist to ClickHouse if repository writer is configured
	if d.writer != nil {
		if err := d.writer.Insert(ctx, anomaly); err != nil {
			observability.DbErrorsTotal.WithLabelValues("clickhouse", "InsertAnomaly").Inc()
			return nil, fmt.Errorf("failed to persist detected anomaly: %w", err)
		}
	}

	return anomaly, nil
}

// ProcessBatch evaluates batches of spans, groups them by (OrgID, TraceID),
// updates baseline statistics, and records any detected anomalies.
func (d *AnomalyDetector) ProcessBatch(ctx context.Context, spans []domain.Span) ([]*domain.Anomaly, error) {
	if len(spans) == 0 {
		return nil, nil
	}

	// Group spans by OrgID and TraceID
	type traceKey struct {
		OrgID   string
		TraceID string
	}
	traces := make(map[traceKey][]domain.Span)

	for _, s := range spans {
		key := traceKey{OrgID: s.OrgID, TraceID: s.TraceID}
		traces[key] = append(traces[key], s)
	}

	var detected []*domain.Anomaly

	for key, traceSpans := range traces {
		anomaly, err := d.AnalyzeTrace(ctx, key.OrgID, traceSpans)
		if err != nil {
			return nil, err
		}
		if anomaly != nil {
			detected = append(detected, anomaly)
		}

		// Update running baseline for every span in trace
		for _, s := range traceSpans {
			d.baseline.Update(s.OrgID, s.ServiceName, s.OperationName, float64(s.DurationMs))
		}
	}

	return detected, nil
}

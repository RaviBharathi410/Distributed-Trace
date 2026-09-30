package analysis

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
)

type ServiceKey struct {
	OrgID         string
	ServiceName   string
	OperationName string
}

type WelfordAccumulator struct {
	Count int64
	Mean  float64
	M2    float64
}

func (w *WelfordAccumulator) Update(x float64) {
	w.Count++
	delta := x - w.Mean
	w.Mean += delta / float64(w.Count)
	delta2 := x - w.Mean
	w.M2 += delta * delta2
}

func (w *WelfordAccumulator) Variance() float64 {
	if w.Count < 2 {
		return 0.0
	}
	return w.M2 / float64(w.Count-1)
}

func (w *WelfordAccumulator) StdDev() float64 {
	return math.Sqrt(w.Variance())
}

type BaselineCalculator struct {
	mu           sync.RWMutex
	baselines    map[ServiceKey]*WelfordAccumulator
	minSamples   int64
	minDuration  float64
	minDeltaMs   float64
	minStdDev    float64
}

type BaselineOptions struct {
	MinSamples  int64
	MinDuration float64
	MinDeltaMs  float64
	MinStdDev   float64
}

func NewBaselineCalculator(opts ...BaselineOptions) *BaselineCalculator {
	opt := BaselineOptions{
		MinSamples:  5,
		MinDuration: 50.0,  // minimum 50ms to alert (filters microsecond noise)
		MinDeltaMs:  30.0,  // observed must exceed baseline by at least 30ms
		MinStdDev:   1.0,   // floor standard deviation at 1.0ms to avoid divide-by-zero
	}
	if len(opts) > 0 {
		if opts[0].MinSamples > 0 {
			opt.MinSamples = opts[0].MinSamples
		}
		if opts[0].MinDuration > 0 {
			opt.MinDuration = opts[0].MinDuration
		}
		if opts[0].MinDeltaMs > 0 {
			opt.MinDeltaMs = opts[0].MinDeltaMs
		}
		if opts[0].MinStdDev > 0 {
			opt.MinStdDev = opts[0].MinStdDev
		}
	}

	return &BaselineCalculator{
		baselines:   make(map[ServiceKey]*WelfordAccumulator),
		minSamples:  opt.MinSamples,
		minDuration: opt.MinDuration,
		minDeltaMs:  opt.MinDeltaMs,
		minStdDev:   opt.MinStdDev,
	}
}

func (b *BaselineCalculator) Update(orgID, serviceName, operationName string, durationMs float64) {
	key := ServiceKey{
		OrgID:         orgID,
		ServiceName:   serviceName,
		OperationName: operationName,
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	acc, ok := b.baselines[key]
	if !ok {
		acc = &WelfordAccumulator{}
		b.baselines[key] = acc
	}
	acc.Update(durationMs)
}

func (b *BaselineCalculator) SetBaseline(orgID, serviceName, operationName string, mean, stdDev float64, count int64) {
	key := ServiceKey{
		OrgID:         orgID,
		ServiceName:   serviceName,
		OperationName: operationName,
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	var m2 float64
	if count > 1 {
		m2 = stdDev * stdDev * float64(count-1)
	}

	b.baselines[key] = &WelfordAccumulator{
		Count: count,
		Mean:  mean,
		M2:    m2,
	}
}

func (b *BaselineCalculator) GetBaseline(orgID, serviceName, operationName string) (mean, stdDev float64, count int64, exists bool) {
	key := ServiceKey{
		OrgID:         orgID,
		ServiceName:   serviceName,
		OperationName: operationName,
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	acc, ok := b.baselines[key]
	if !ok || acc.Count == 0 {
		return 0, 0, 0, false
	}

	sd := acc.StdDev()
	if sd < b.minStdDev {
		sd = b.minStdDev
	}

	return acc.Mean, sd, acc.Count, true
}

func (b *BaselineCalculator) CalculateZScore(orgID, serviceName, operationName string, observedMs float64) (zScore float64, baselineMean float64, isAnomalous bool) {
	key := ServiceKey{
		OrgID:         orgID,
		ServiceName:   serviceName,
		OperationName: operationName,
	}

	b.mu.RLock()
	acc, ok := b.baselines[key]
	b.mu.RUnlock()

	if !ok || acc.Count < b.minSamples {
		// Insufficient samples to establish reliable baseline (prevents cold-start false positives)
		return 0, observedMs, false
	}

	mean := acc.Mean
	sd := acc.StdDev()
	if sd < b.minStdDev {
		sd = b.minStdDev
	}

	delta := observedMs - mean
	if delta <= 0 {
		// Performance faster than or equal to baseline is not a latency anomaly
		return 0, mean, false
	}

	zScore = delta / sd

	// Criteria for anomaly:
	// 1. Z >= 3.0 (standard 3-sigma statistical threshold)
	// 2. Observed duration >= MinDuration (filters microsecond operations)
	// 3. Absolute delta >= MinDeltaMs (filters small shifts like 2ms -> 5ms with tiny variance)
	if zScore >= 3.0 && observedMs >= b.minDuration && delta >= b.minDeltaMs {
		return zScore, mean, true
	}

	return zScore, mean, false
}

// PreloadFromClickHouse seeds the baseline accumulator from historical span metrics in ClickHouse.
func (b *BaselineCalculator) PreloadFromClickHouse(ctx context.Context, conn clickhouse.Conn, orgID string, since time.Time) error {
	reqID := observability.GetRequestID(ctx)
	comment := ""
	if reqID != "" {
		comment = "/* request_id=" + reqID + " */"
	}

	query := fmt.Sprintf(`
		SELECT
			service_name,
			operation_name,
			avg(duration_ms) as mean_duration,
			stddevSamp(duration_ms) as stddev_duration,
			count() as sample_count
		FROM otel_spans
		WHERE org_id = ? AND start_time >= ?
		GROUP BY service_name, operation_name
		HAVING sample_count >= ?
		%s`, comment)

	rows, err := conn.Query(ctx, query, orgID, since, b.minSamples)
	if err != nil {
		return fmt.Errorf("failed to query historical baselines from clickhouse: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var serviceName, operationName string
		var mean, stddev float64
		var count uint64

		if err := rows.Scan(&serviceName, &operationName, &mean, &stddev, &count); err != nil {
			return fmt.Errorf("failed to scan baseline row: %w", err)
		}

		b.SetBaseline(orgID, serviceName, operationName, mean, stddev, int64(count))
	}

	return nil
}

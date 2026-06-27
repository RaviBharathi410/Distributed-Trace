package domain

import "time"

type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type ServiceStats struct {
	ServiceName   string    `json:"service_name"`
	P50Ms         float64   `json:"p50"`
	P95Ms         float64   `json:"p95"`
	P99Ms         float64   `json:"p99"`
	ErrorRatePct  float64   `json:"error_rate"`
	RequestRate   float64   `json:"request_rate"` // RPS
	TimeRangeFrom time.Time `json:"from"`
	TimeRangeTo   time.Time `json:"to"`
}

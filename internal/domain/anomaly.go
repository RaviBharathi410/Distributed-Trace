package domain

import "time"

type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

type Anomaly struct {
	ID                 string    `json:"id"`
	OrgID              string    `json:"org_id"`
	ServiceName        string    `json:"service_name"`
	OperationName      string    `json:"operation_name"`
	RootCauseService   string    `json:"root_cause_service"`
	Severity           Severity  `json:"severity"`
	ZScore             float64   `json:"z_score"`
	BaselineLatencyMs  float64   `json:"baseline_latency_ms"`
	ObservedLatencyMs  float64   `json:"observed_latency_ms"`
	ErrorRateDelta     float64   `json:"error_rate_delta"` // percentage change
	Status             string    `json:"status"`           // "open", "investigating", "resolved"
	DetectedAt         time.Time `json:"detected_at"`
	ResolvedAt         *time.Time `json:"resolved_at,omitempty"`
	RootCausePath      []string  `json:"root_cause_path"`
}

type AnomalyFilter struct {
	Severity string `json:"severity"` // "all" or specific
	Service  string `json:"service"`
	Status   string `json:"status"`   // "all", "open", "investigating", "resolved"
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
}

type AnomalyStats struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

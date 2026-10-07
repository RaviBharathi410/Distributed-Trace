package domain

import "time"

// ExplanationCost encapsulates token counts, calculated charges, and budget ceilings.
type ExplanationCost struct {
	Model            string  `json:"model"`
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
	Cached           bool    `json:"cached"`
	CostCeilingUSD   float64 `json:"cost_ceiling_usd"`
	HourlyCapUSD     float64 `json:"hourly_cap_usd"`
	HourlySpendUSD   float64 `json:"hourly_spend_usd"`
}

// IncidentExplanation represents an on-demand, read-only root cause explanation for an anomaly.
type IncidentExplanation struct {
	AnomalyID               string          `json:"anomaly_id"`
	OrgID                   string          `json:"org_id"`
	RootCauseService        string          `json:"root_cause_service"`
	OperationName           string          `json:"operation_name"`
	ConfidenceScore         float64         `json:"confidence_score"`
	Summary                 string          `json:"summary"`
	ContributingFactors     []string        `json:"contributing_factors"`
	DegradedToDeterministic bool            `json:"degraded_to_deterministic"`
	FallbackReason          string          `json:"fallback_reason,omitempty"`
	CostAttribution         ExplanationCost `json:"cost_attribution"`
	GeneratedAt             time.Time       `json:"generated_at"`
}

// ExplainAnomalyRequest specifies optional parameters for the on-demand explanation endpoint.
type ExplainAnomalyRequest struct {
	ForceRefresh bool `json:"force_refresh"`
}

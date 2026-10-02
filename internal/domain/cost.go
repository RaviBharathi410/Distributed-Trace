package domain

import "time"

type LLMCostEvent struct {
	ID               string    `json:"id"`
	OrgID            string    `json:"orgId"`
	AnomalyID        string    `json:"anomalyId"`
	Model            string    `json:"model"`
	InputTokens      int       `json:"inputTokens"`
	OutputTokens     int       `json:"outputTokens"`
	EstimatedCostUSD float64   `json:"estimatedCostUsd"`
	CreatedAt        time.Time `json:"createdAt"`
}

type CostSummary struct {
	OrgID                   string  `json:"orgId"`
	InfraShareUSD           float64 `json:"infraShareUsd"`
	LLMSpendUSD             float64 `json:"llmSpendUsd"`
	TotalCostUSD            float64 `json:"totalCostUsd"`
	TotalIncidentsExplained int     `json:"totalIncidentsExplained"`
	AvgCostPerIncidentUSD   float64 `json:"avgCostPerIncidentUsd"`
	HourlySpendUSD          float64 `json:"hourlySpendUsd"`
	HourlyCapUSD            float64 `json:"hourlyCapUsd"`
	ActiveOrgsCount         int     `json:"activeOrgsCount"`
}

type CostBreakdownItem struct {
	ID               string    `json:"id"`
	AnomalyID        string    `json:"anomalyId"`
	Model            string    `json:"model"`
	InputTokens      int       `json:"inputTokens"`
	OutputTokens     int       `json:"outputTokens"`
	EstimatedCostUSD float64   `json:"estimatedCostUsd"`
	CreatedAt        time.Time `json:"createdAt"`
}

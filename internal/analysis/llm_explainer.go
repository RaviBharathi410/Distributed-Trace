package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
	"go.uber.org/zap"
)

var (
	ErrNoAPIKey = errors.New("LLM API key not configured")
)

// LLMClient is the pluggable interface for issuing diagnostic generation requests.
type LLMClient interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string) (rawContent string, inputTokens, outputTokens int, err error)
}

// CostRecorder records cost events and calculates tenant hourly spend for circuit breakers.
type CostRecorder interface {
	RecordCost(ctx context.Context, event *domain.LLMCostEvent) error
	GetHourlySpend(ctx context.Context, orgID string) (float64, error)
}

// ExplainerConfig defines pricing parameters and circuit breaker thresholds.
type ExplainerConfig struct {
	ModelName              string
	InputPricePerMillion   float64
	OutputPricePerMillion  float64
	CostCeilingPerIncident float64
	HourlySpendCap         float64
}

// GeminiClient implements LLMClient for Google's Gemini REST API.
type GeminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewGeminiClient(apiKey, model string) *GeminiClient {
	if model == "" {
		model = "gemini-1.5-flash"
	}
	return &GeminiClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiGenConfig struct {
	Temperature      float64 `json:"temperature"`
	MaxOutputTokens  int     `json:"maxOutputTokens"`
	ResponseMimeType string  `json:"responseMimeType"`
}

type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  geminiGenConfig `json:"generationConfig"`
}

type geminiCandidate struct {
	Content struct {
		Parts []geminiPart `json:"parts"`
	} `json:"content"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type geminiResponse struct {
	Candidates    []geminiCandidate   `json:"candidates"`
	UsageMetadata geminiUsageMetadata `json:"usageMetadata"`
	Error         *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

func (c *GeminiClient) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, int, int, error) {
	if c.apiKey == "" {
		return "", 0, 0, ErrNoAPIKey
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)

	reqPayload := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		},
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: userPrompt}},
			},
		},
		GenerationConfig: geminiGenConfig{
			Temperature:      0.2,
			MaxOutputTokens:  350,
			ResponseMimeType: "application/json",
		},
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to marshal gemini request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", 0, 0, fmt.Errorf("gemini request error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to read gemini response body: %w", err)
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return "", 0, 0, fmt.Errorf("failed to parse gemini response json: %w", err)
	}

	if geminiResp.Error != nil {
		return "", 0, 0, fmt.Errorf("gemini api error (code %d, status %s): %s",
			geminiResp.Error.Code, geminiResp.Error.Status, geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", 0, 0, errors.New("empty response candidates from gemini")
	}

	rawContent := geminiResp.Candidates[0].Content.Parts[0].Text
	inTokens := geminiResp.UsageMetadata.PromptTokenCount
	outTokens := geminiResp.UsageMetadata.CandidatesTokenCount

	return rawContent, inTokens, outTokens, nil
}

type cachedExplanation struct {
	explanation domain.IncidentExplanation
	cachedAt    time.Time
}

// IncidentExplainer coordinates prompt construction, pre-flight budget checks,
// storm spend verification, LLM execution, two-tier output validation, and 24h caching.
type IncidentExplainer struct {
	client    LLMClient
	validator *OutputValidator
	costs     CostRecorder
	cfg       ExplainerConfig
	cache     map[string]cachedExplanation
	cacheMu   sync.RWMutex
}

func NewIncidentExplainer(client LLMClient, validator *OutputValidator, costs CostRecorder, cfg ExplainerConfig) *IncidentExplainer {
	if validator == nil {
		validator = NewOutputValidator()
	}
	return &IncidentExplainer{
		client:    client,
		validator: validator,
		costs:     costs,
		cfg:       cfg,
		cache:     make(map[string]cachedExplanation),
	}
}

const systemPrompt = `You are a read-only root-cause diagnostic engine for distributed traces.
Your SOLE task is to explain what occurred based strictly on the provided telemetry facts.

CRITICAL SAFETY & SCOPE RULES:
1. Do NOT volunteer or suggest remediation actions, operational fixes, or configuration changes.
2. Do NOT use soft advisory phrasing (e.g., "this pattern is often resolved by...", "typically indicates the service needs more replicas", "consider tuning...").
3. Do NOT generate shell commands, kubectl directives, or infrastructure mutation instructions.
4. Confine your response strictly to descriptive diagnosis of observed telemetry anomalies and contributing factors.
5. Output must be valid JSON matching the exact schema provided.

CONTRASTIVE EXAMPLES:

❌ VIOLATION (Soft Advisory / Prescriptive Phrasing - FORBIDDEN):
{
  "summary": "This pattern is often resolved by increasing connection pool size on payment-svc.",
  "contributing_factors": [
    "Typically indicates the service needs more replicas to handle traffic bursts.",
    "This is commonly addressed by a cache warm restart."
  ]
}

✅ COMPLIANT (Descriptive Observation of Telemetry Only - REQUIRED):
{
  "summary": "Database connection acquisition wait time in 'payment-svc' spiked from 15ms to 240ms, causing request queue buildup.",
  "contributing_factors": [
    "Active connections reached the configured pool limit of 100 at 14:02:10 UTC.",
    "Upstream edge 'checkout-api' -> 'payment-svc' observed 260ms p95 latency while error rate remained at 0.4%."
  ],
  "confidence_score": 0.94
}`

// BuildUserPrompt formats deterministic Phase 3 outputs into a grounded prompt.
func BuildUserPrompt(anomaly *domain.Anomaly) string {
	pathStr := "None recorded"
	if len(anomaly.RootCausePath) > 0 {
		pathStr = strings.Join(anomaly.RootCausePath, " -> ")
	}

	return fmt.Sprintf(`INCIDENT TELEMETRY FACTS (DETERMINISTIC GROUND TRUTH):
- Anomaly ID: %s
- Root Cause Service: %s
- Affected Service: %s
- Endpoint / Operation: %s
- Observed Latency: %.2f ms
- Baseline Latency: %.2f ms (7-day rolling Welford mean)
- Latency Delta: +%.2f ms
- Z-Score: %.2f (Bessel-corrected sample standard deviations above baseline)
- Error Rate Delta: +%.2f%%
- Critical Path Bottleneck Traversal: %s
- Severity: %s
- Detected At: %s

Generate a descriptive diagnosis JSON object matching:
{
  "summary": "Descriptive summary of the telemetry spike",
  "contributing_factors": [
    "Factor 1",
    "Factor 2"
  ],
  "confidence_score": 0.95
}`,
		anomaly.ID,
		anomaly.RootCauseService,
		anomaly.ServiceName,
		anomaly.OperationName,
		anomaly.ObservedLatencyMs,
		anomaly.BaselineLatencyMs,
		anomaly.ObservedLatencyMs-anomaly.BaselineLatencyMs,
		anomaly.ZScore,
		anomaly.ErrorRateDelta,
		pathStr,
		anomaly.Severity,
		anomaly.DetectedAt.UTC().Format(time.RFC3339),
	)
}

// BuildDeterministicFallback constructs a structured mathematical telemetry fallback
// directly from Phase 3 detection data with zero token fees and zero operational risk.
func (e *IncidentExplainer) BuildDeterministicFallback(anomaly *domain.Anomaly, reason string, hourlySpend float64, postRejectionCost *domain.ExplanationCost) *domain.IncidentExplanation {
	pathStr := anomaly.RootCauseService
	if len(anomaly.RootCausePath) > 0 {
		pathStr = strings.Join(anomaly.RootCausePath, " -> ")
	}

	summary := fmt.Sprintf("Deterministic telemetry anomaly isolated in service '%s' during operation '%s'. Observed latency of %.1fms exceeded historical Welford baseline of %.1fms (Z-score: %.2f, error rate delta: +%.1f%%).",
		anomaly.RootCauseService, anomaly.OperationName, anomaly.ObservedLatencyMs, anomaly.BaselineLatencyMs, anomaly.ZScore, anomaly.ErrorRateDelta)

	factors := []string{
		fmt.Sprintf("Observed latency: %.1fms vs 7-day Welford rolling baseline: %.1fms (Z-Score: %.2f).", anomaly.ObservedLatencyMs, anomaly.BaselineLatencyMs, anomaly.ZScore),
		fmt.Sprintf("Critical path bottleneck traversal: %s.", pathStr),
		"Pure descriptive telemetry observation ($0 LLM fee incurred); zero remediation directives permitted per Phase 5A safety boundary.",
	}

	costAttr := domain.ExplanationCost{
		Model:            "deterministic-fallback",
		InputTokens:      0,
		OutputTokens:     0,
		EstimatedCostUSD: 0.0,
		Cached:           false,
		CostCeilingUSD:   e.cfg.CostCeilingPerIncident,
		HourlyCapUSD:     e.cfg.HourlySpendCap,
		HourlySpendUSD:   hourlySpend,
	}

	// If degraded post-generation due to validator rejection, retain the honest recorded token cost
	if postRejectionCost != nil {
		costAttr.Model = postRejectionCost.Model
		costAttr.InputTokens = postRejectionCost.InputTokens
		costAttr.OutputTokens = postRejectionCost.OutputTokens
		costAttr.EstimatedCostUSD = postRejectionCost.EstimatedCostUSD
	}

	return &domain.IncidentExplanation{
		AnomalyID:               anomaly.ID,
		OrgID:                   anomaly.OrgID,
		RootCauseService:        anomaly.RootCauseService,
		OperationName:           anomaly.OperationName,
		ConfidenceScore:         1.0, // Mathematical certainty from deterministic data
		Summary:                 summary,
		ContributingFactors:     factors,
		DegradedToDeterministic: true,
		FallbackReason:          reason,
		CostAttribution:         costAttr,
		GeneratedAt:             time.Now().UTC(),
	}
}

type llmOutputSchema struct {
	Summary             string   `json:"summary"`
	ContributingFactors []string `json:"contributing_factors"`
	ConfidenceScore     float64  `json:"confidence_score"`
}

// Explain analyzes an anomaly and generates a read-only root cause explanation.
func (e *IncidentExplainer) Explain(ctx context.Context, orgID string, anomaly *domain.Anomaly, forceRefresh bool) (*domain.IncidentExplanation, error) {
	if anomaly == nil {
		return nil, errors.New("anomaly cannot be nil")
	}

	cacheKey := fmt.Sprintf("%s:%s", orgID, anomaly.ID)

	// 1. Check 24-hour idempotent cache
	if !forceRefresh {
		e.cacheMu.RLock()
		item, found := e.cache[cacheKey]
		e.cacheMu.RUnlock()

		if found && time.Since(item.cachedAt) < 24*time.Hour {
			cachedExp := item.explanation
			cachedExp.CostAttribution.Cached = true
			cachedExp.CostAttribution.EstimatedCostUSD = 0.0
			cachedExp.CostAttribution.InputTokens = 0
			cachedExp.CostAttribution.OutputTokens = 0

			// Dynamically populate current hourly spend status
			if e.costs != nil {
				if hSpend, err := e.costs.GetHourlySpend(ctx, orgID); err == nil {
					cachedExp.CostAttribution.HourlySpendUSD = hSpend
				}
			}
			return &cachedExp, nil
		}
	}

	// 2. Query tenant current hourly spend for storm circuit breaker
	var hourlySpend float64
	if e.costs != nil {
		var err error
		hourlySpend, err = e.costs.GetHourlySpend(ctx, orgID)
		if err != nil {
			observability.Log.Warn("Failed to fetch tenant hourly spend; defaulting to 0", zap.String("org_id", orgID), zap.Error(err))
		}
	}

	// Check Hourly Storm Spend Cap ($1.00/hr)
	if e.cfg.HourlySpendCap > 0 && hourlySpend >= e.cfg.HourlySpendCap {
		reason := fmt.Sprintf("Tenant hourly storm spend cap ($%.2f/hr) reached (current: $%.4f/hr); short-circuited to prevent cascading token spend",
			e.cfg.HourlySpendCap, hourlySpend)
		fallback := e.BuildDeterministicFallback(anomaly, reason, hourlySpend, nil)
		return fallback, nil
	}

	// 3. Pre-flight token calculation & Cost Ceiling Enforcement (≤ $0.01 per incident)
	userPrompt := BuildUserPrompt(anomaly)
	// Standard heuristic: 4 characters per token
	estInputTokens := (len(systemPrompt) + len(userPrompt) + 3) / 4
	const maxOutputTokens = 350
	estCost := (float64(estInputTokens)*e.cfg.InputPricePerMillion + float64(maxOutputTokens)*e.cfg.OutputPricePerMillion) / 1_000_000.0

	if e.cfg.CostCeilingPerIncident > 0 && estCost > e.cfg.CostCeilingPerIncident {
		reason := fmt.Sprintf("Pre-flight token cost estimate ($%.6f) exceeded incident ceiling ($%.6f); short-circuited before provider dispatch",
			estCost, e.cfg.CostCeilingPerIncident)
		fallback := e.BuildDeterministicFallback(anomaly, reason, hourlySpend, nil)
		return fallback, nil
	}

	// 4. Verify LLM client availability
	if e.client == nil {
		reason := "LLM client unconfigured or unavailable; degraded to deterministic telemetry observation"
		return e.BuildDeterministicFallback(anomaly, reason, hourlySpend, nil), nil
	}

	// 5. Invoke LLM provider
	rawResponse, inTokens, outTokens, err := e.client.Generate(ctx, systemPrompt, userPrompt)
	if err != nil {
		if errors.Is(err, ErrNoAPIKey) {
			reason := "LLM provider unconfigured (API key missing); served deterministic telemetry observation"
			return e.BuildDeterministicFallback(anomaly, reason, hourlySpend, nil), nil
		}
		observability.Log.Warn("LLM generation request failed; degrading to deterministic fallback",
			zap.String("anomaly_id", anomaly.ID), zap.Error(err))
		reason := fmt.Sprintf("LLM provider dispatch failed (%v); degraded to deterministic telemetry observation", err)
		return e.BuildDeterministicFallback(anomaly, reason, hourlySpend, nil), nil
	}

	// Calculate actual vendor cost from observed tokens
	actualCost := (float64(inTokens)*e.cfg.InputPricePerMillion + float64(outTokens)*e.cfg.OutputPricePerMillion) / 1_000_000.0

	// 6. Record token usage to cost ledger (Mandatory for honesty even if rejected)
	if e.costs != nil {
		costEvent := &domain.LLMCostEvent{
			OrgID:            orgID,
			AnomalyID:        anomaly.ID,
			Model:            e.cfg.ModelName,
			InputTokens:      inTokens,
			OutputTokens:     outTokens,
			EstimatedCostUSD: actualCost,
			CreatedAt:        time.Now().UTC(),
		}
		if recErr := e.costs.RecordCost(ctx, costEvent); recErr != nil {
			observability.Log.Error("Failed to record tenant LLM cost event", zap.String("org_id", orgID), zap.Error(recErr))
		}
	}

	postRejectionCost := &domain.ExplanationCost{
		Model:            e.cfg.ModelName,
		InputTokens:      inTokens,
		OutputTokens:     outTokens,
		EstimatedCostUSD: actualCost,
	}

	// 7. Parse JSON structure
	var parsed llmOutputSchema
	if unmarshalErr := json.Unmarshal([]byte(rawResponse), &parsed); unmarshalErr != nil {
		reason := fmt.Sprintf("LLM output did not adhere to required JSON schema (%v); degraded to deterministic telemetry observation", unmarshalErr)
		return e.BuildDeterministicFallback(anomaly, reason, hourlySpend+actualCost, postRejectionCost), nil
	}

	// 8. Two-Tiered Output Validation (Hard stop — zero re-prompts)
	valErr := e.validator.ValidateDiagnosticOutput(parsed.Summary, parsed.ContributingFactors)
	if valErr != nil {
		reason := fmt.Sprintf("Safety validator rejected generated output (%v); degraded to deterministic telemetry observation", valErr)
		observability.Log.Warn("Output validator rejected LLM diagnostic output",
			zap.String("anomaly_id", anomaly.ID),
			zap.Error(valErr),
		)
		return e.BuildDeterministicFallback(anomaly, reason, hourlySpend+actualCost, postRejectionCost), nil
	}

	// 9. Validated explanation assembly
	confScore := parsed.ConfidenceScore
	if confScore <= 0.0 || confScore > 1.0 {
		confScore = 0.95
	}

	explanation := &domain.IncidentExplanation{
		AnomalyID:               anomaly.ID,
		OrgID:                   orgID,
		RootCauseService:        anomaly.RootCauseService,
		OperationName:           anomaly.OperationName,
		ConfidenceScore:         confScore,
		Summary:                 parsed.Summary,
		ContributingFactors:     parsed.ContributingFactors,
		DegradedToDeterministic: false,
		CostAttribution: domain.ExplanationCost{
			Model:            e.cfg.ModelName,
			InputTokens:      inTokens,
			OutputTokens:     outTokens,
			EstimatedCostUSD: actualCost,
			Cached:           false,
			CostCeilingUSD:   e.cfg.CostCeilingPerIncident,
			HourlyCapUSD:     e.cfg.HourlySpendCap,
			HourlySpendUSD:   hourlySpend + actualCost,
		},
		GeneratedAt: time.Now().UTC(),
	}

	// 10. Store in 24h cache
	e.cacheMu.Lock()
	e.cache[cacheKey] = cachedExplanation{
		explanation: *explanation,
		cachedAt:    time.Now().UTC(),
	}
	e.cacheMu.Unlock()

	return explanation, nil
}

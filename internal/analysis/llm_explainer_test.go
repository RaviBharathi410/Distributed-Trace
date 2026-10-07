package analysis

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
)

type mockLLMClient struct {
	calls        int
	mu           sync.Mutex
	response     string
	inputTokens  int
	outputTokens int
	err          error
}

func (m *mockLLMClient) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, int, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.err != nil {
		return "", 0, 0, m.err
	}
	return m.response, m.inputTokens, m.outputTokens, nil
}

type mockCostRecorder struct {
	mu           sync.Mutex
	recorded     []*domain.LLMCostEvent
	hourlySpend  float64
	hourlyErr    error
	recordErr    error
}

func (m *mockCostRecorder) RecordCost(ctx context.Context, event *domain.LLMCostEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordErr != nil {
		return m.recordErr
	}
	m.recorded = append(m.recorded, event)
	return nil
}

func (m *mockCostRecorder) GetHourlySpend(ctx context.Context, orgID string) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.hourlySpend, m.hourlyErr
}

func sampleAnomaly() *domain.Anomaly {
	return &domain.Anomaly{
		ID:                "anm-test-01",
		OrgID:             "org-123",
		ServiceName:       "checkout-api",
		OperationName:     "POST /order",
		RootCauseService:  "payment-svc",
		Severity:          domain.SeverityCritical,
		ZScore:            4.85,
		BaselineLatencyMs: 25.0,
		ObservedLatencyMs: 210.0,
		ErrorRateDelta:    1.2,
		Status:            "open",
		DetectedAt:        time.Now().UTC().Add(-5 * time.Minute),
		RootCausePath:     []string{"checkout-api", "payment-svc", "postgres-db"},
	}
}

func sampleExplainerConfig() ExplainerConfig {
	return ExplainerConfig{
		ModelName:              "gemini-1.5-flash",
		InputPricePerMillion:   0.075,
		OutputPricePerMillion:  0.300,
		CostCeilingPerIncident: 0.010,
		HourlySpendCap:         1.000,
	}
}

func TestIncidentExplainer_PreFlightCostCeiling_DegradesToDeterministic(t *testing.T) {
	mockClient := &mockLLMClient{}
	mockCosts := &mockCostRecorder{}
	cfg := sampleExplainerConfig()
	// Set an impossible cost ceiling so pre-flight check triggers immediately
	cfg.CostCeilingPerIncident = 0.0000001

	explainer := NewIncidentExplainer(mockClient, nil, mockCosts, cfg)
	anomaly := sampleAnomaly()

	res, err := explainer.Explain(context.Background(), "org-123", anomaly, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.DegradedToDeterministic {
		t.Errorf("expected DegradedToDeterministic to be true when pre-flight ceiling is exceeded")
	}
	if !strings.Contains(res.FallbackReason, "Pre-flight token cost estimate") {
		t.Errorf("expected fallback reason to cite pre-flight token estimate, got: %s", res.FallbackReason)
	}
	if mockClient.calls != 0 {
		t.Errorf("expected 0 LLM client calls on pre-flight rejection, got %d", mockClient.calls)
	}
	if res.CostAttribution.EstimatedCostUSD != 0.0 {
		t.Errorf("expected $0.00 cost attribution on pre-flight fallback, got %f", res.CostAttribution.EstimatedCostUSD)
	}
}

func TestIncidentExplainer_HourlyStormSpendCap_DegradesToDeterministic(t *testing.T) {
	mockClient := &mockLLMClient{}
	mockCosts := &mockCostRecorder{hourlySpend: 1.25} // Already exceeded $1.00/hr
	cfg := sampleExplainerConfig()

	explainer := NewIncidentExplainer(mockClient, nil, mockCosts, cfg)
	anomaly := sampleAnomaly()

	res, err := explainer.Explain(context.Background(), "org-123", anomaly, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.DegradedToDeterministic {
		t.Errorf("expected DegradedToDeterministic to be true when hourly cap is exceeded")
	}
	if !strings.Contains(res.FallbackReason, "hourly storm spend cap") {
		t.Errorf("expected fallback reason to cite storm spend cap, got: %s", res.FallbackReason)
	}
	if mockClient.calls != 0 {
		t.Errorf("expected 0 LLM client calls when storm spend cap tripped, got %d", mockClient.calls)
	}
}

func TestIncidentExplainer_ValidGeneration_PassesValidationAndCaches(t *testing.T) {
	validJSON := `{
		"summary": "Database connection acquisition wait time in payment-svc spiked from 15ms to 240ms, causing request queue buildup.",
		"contributing_factors": [
			"Active connections reached the configured pool limit of 100 at 14:02:10 UTC.",
			"Upstream edge checkout-api -> payment-svc observed 260ms p95 latency while error rate remained at 0.4%."
		],
		"confidence_score": 0.94
	}`

	mockClient := &mockLLMClient{
		response:     validJSON,
		inputTokens:  850,
		outputTokens: 150,
	}
	mockCosts := &mockCostRecorder{}
	cfg := sampleExplainerConfig()

	explainer := NewIncidentExplainer(mockClient, nil, mockCosts, cfg)
	anomaly := sampleAnomaly()

	// First call — generates and caches
	res, err := explainer.Explain(context.Background(), "org-123", anomaly, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.DegradedToDeterministic {
		t.Errorf("expected DegradedToDeterministic to be false for valid descriptive output")
	}
	if res.ConfidenceScore != 0.94 {
		t.Errorf("expected confidence score 0.94, got %f", res.ConfidenceScore)
	}
	if len(res.ContributingFactors) != 2 {
		t.Errorf("expected 2 contributing factors, got %d", len(res.ContributingFactors))
	}
	if mockClient.calls != 1 {
		t.Errorf("expected 1 LLM client call, got %d", mockClient.calls)
	}
	if len(mockCosts.recorded) != 1 {
		t.Errorf("expected 1 cost event recorded in ledger, got %d", len(mockCosts.recorded))
	}

	// Verify cost calculation: (850*0.075 + 150*0.300) / 1,000,000 = (63.75 + 45) / 1,000,000 = 0.00010875
	expectedCost := (850.0*0.075 + 150.0*0.300) / 1_000_000.0
	if res.CostAttribution.EstimatedCostUSD != expectedCost {
		t.Errorf("expected cost %f, got %f", expectedCost, res.CostAttribution.EstimatedCostUSD)
	}

	// Second call — returns from 24h cache
	cachedRes, err := explainer.Explain(context.Background(), "org-123", anomaly, false)
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}
	if !cachedRes.CostAttribution.Cached {
		t.Errorf("expected cachedRes to be marked cached")
	}
	if cachedRes.CostAttribution.EstimatedCostUSD != 0.0 {
		t.Errorf("expected $0.00 cost for cached explanation, got %f", cachedRes.CostAttribution.EstimatedCostUSD)
	}
	if mockClient.calls != 1 {
		t.Errorf("expected still only 1 client call after cached retrieval, got %d", mockClient.calls)
	}

	// Third call — force refresh bypasses cache
	refreshedRes, err := explainer.Explain(context.Background(), "org-123", anomaly, true)
	if err != nil {
		t.Fatalf("unexpected error on force refresh: %v", err)
	}
	if refreshedRes.CostAttribution.Cached {
		t.Errorf("expected forceRefresh to bypass cache")
	}
	if mockClient.calls != 2 {
		t.Errorf("expected 2 client calls after force refresh, got %d", mockClient.calls)
	}
}

func TestIncidentExplainer_ValidatorRejection_HardStopZeroRePrompts_DegradesToDeterministic(t *testing.T) {
	// Adversarial output volunteering soft advisory and remediation commands
	adversarialJSON := `{
		"summary": "This pattern is often resolved by increasing connection pool size on payment-svc.",
		"contributing_factors": [
			"Typically indicates the service needs more replicas to handle current ingress.",
			"Run kubectl scale deployment payment-svc --replicas=5 to resolve."
		],
		"confidence_score": 0.90
	}`

	mockClient := &mockLLMClient{
		response:     adversarialJSON,
		inputTokens:  900,
		outputTokens: 200,
	}
	mockCosts := &mockCostRecorder{}
	cfg := sampleExplainerConfig()

	explainer := NewIncidentExplainer(mockClient, nil, mockCosts, cfg)
	anomaly := sampleAnomaly()

	res, err := explainer.Explain(context.Background(), "org-123", anomaly, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.DegradedToDeterministic {
		t.Errorf("expected DegradedToDeterministic to be true when validator rejects output")
	}
	if !strings.Contains(res.FallbackReason, "Safety validator rejected generated output") {
		t.Errorf("expected fallback reason to cite safety validator, got: %s", res.FallbackReason)
	}

	// CRITICAL TEST: Zero re-prompts!
	if mockClient.calls != 1 {
		t.Fatalf("CRITICAL SAFETY VIOLATION: expected exactly 1 call (ZERO re-prompts on validator rejection), got %d", mockClient.calls)
	}

	// CRITICAL TEST: Honest token tracking even for rejected output
	if len(mockCosts.recorded) != 1 {
		t.Fatalf("expected rejected call tokens to still be recorded to cost ledger, got %d events", len(mockCosts.recorded))
	}
	if mockCosts.recorded[0].InputTokens != 900 || mockCosts.recorded[0].OutputTokens != 200 {
		t.Errorf("expected recorded tokens (900, 200), got (%d, %d)", mockCosts.recorded[0].InputTokens, mockCosts.recorded[0].OutputTokens)
	}

	// Fallback cost attribution reflects the rejected call's actual cost for ledger honesty
	expectedCost := (900.0*0.075 + 200.0*0.300) / 1_000_000.0
	if res.CostAttribution.EstimatedCostUSD != expectedCost {
		t.Errorf("expected cost attribution to reflect rejected tokens (%f), got %f", expectedCost, res.CostAttribution.EstimatedCostUSD)
	}
}

func TestIncidentExplainer_UnconfiguredClient_GracefulFallback(t *testing.T) {
	mockClient := &mockLLMClient{err: ErrNoAPIKey}
	mockCosts := &mockCostRecorder{}
	cfg := sampleExplainerConfig()

	explainer := NewIncidentExplainer(mockClient, nil, mockCosts, cfg)
	anomaly := sampleAnomaly()

	res, err := explainer.Explain(context.Background(), "org-123", anomaly, false)
	if err != nil {
		t.Fatalf("unexpected error on unconfigured client: %v", err)
	}

	if !res.DegradedToDeterministic {
		t.Errorf("expected DegradedToDeterministic to be true when API key is missing")
	}
	if !strings.Contains(res.FallbackReason, "LLM provider unconfigured") {
		t.Errorf("expected fallback reason to state API key missing, got: %s", res.FallbackReason)
	}
	if res.CostAttribution.EstimatedCostUSD != 0.0 {
		t.Errorf("expected $0.00 cost when unconfigured, got %f", res.CostAttribution.EstimatedCostUSD)
	}
}

func TestIncidentExplainer_NetworkTimeoutOrConnectionError_DegradesToDeterministic_ZeroCostRecorded(t *testing.T) {
	mockClient := &mockLLMClient{
		err: errors.New("Post \"https://generativelanguage.googleapis.com/... \": context deadline exceeded (Client.Timeout exceeded)"),
	}
	mockCosts := &mockCostRecorder{}
	cfg := sampleExplainerConfig()

	explainer := NewIncidentExplainer(mockClient, nil, mockCosts, cfg)
	anomaly := sampleAnomaly()

	res, err := explainer.Explain(context.Background(), "org-123", anomaly, false)
	if err != nil {
		t.Fatalf("unexpected error during network failure: %v", err)
	}

	if !res.DegradedToDeterministic {
		t.Errorf("expected DegradedToDeterministic to be true on network failure")
	}
	if !strings.Contains(res.FallbackReason, "LLM provider dispatch failed") {
		t.Errorf("expected fallback reason to cite provider dispatch failure, got: %s", res.FallbackReason)
	}
	if !strings.Contains(res.FallbackReason, "context deadline exceeded") {
		t.Errorf("expected fallback reason to preserve root cause error, got: %s", res.FallbackReason)
	}

	// CRITICAL TEST: Zero cost and zero tokens logged when network call aborted
	if res.CostAttribution.EstimatedCostUSD != 0.0 {
		t.Errorf("expected $0.00 cost when network call failed, got %f", res.CostAttribution.EstimatedCostUSD)
	}
	if res.CostAttribution.InputTokens != 0 || res.CostAttribution.OutputTokens != 0 {
		t.Errorf("expected 0 tokens attributed, got in=%d, out=%d", res.CostAttribution.InputTokens, res.CostAttribution.OutputTokens)
	}
	if len(mockCosts.recorded) != 0 {
		t.Errorf("expected 0 cost events recorded in ledger for aborted network call, got %d", len(mockCosts.recorded))
	}
}

func TestIncidentExplainer_Provider5xxError_DegradesToDeterministic_ZeroCostRecorded(t *testing.T) {
	mockClient := &mockLLMClient{
		err: fmt.Errorf("gemini api error (code 503, status UNAVAILABLE): The model is overloaded. Please try again later."),
	}
	mockCosts := &mockCostRecorder{}
	cfg := sampleExplainerConfig()

	explainer := NewIncidentExplainer(mockClient, nil, mockCosts, cfg)
	anomaly := sampleAnomaly()

	res, err := explainer.Explain(context.Background(), "org-123", anomaly, false)
	if err != nil {
		t.Fatalf("unexpected error on 5xx: %v", err)
	}

	if !res.DegradedToDeterministic {
		t.Errorf("expected DegradedToDeterministic to be true on 5xx")
	}
	if !strings.Contains(res.FallbackReason, "UNAVAILABLE") {
		t.Errorf("expected fallback reason to mention UNAVAILABLE, got: %s", res.FallbackReason)
	}
	if res.CostAttribution.EstimatedCostUSD != 0.0 {
		t.Errorf("expected $0.00 cost on 5xx, got %f", res.CostAttribution.EstimatedCostUSD)
	}
	if len(mockCosts.recorded) != 0 {
		t.Errorf("expected 0 cost events recorded in ledger for 5xx, got %d", len(mockCosts.recorded))
	}
}

func TestIncidentExplainer_MalformedJSONResponse_DegradesToDeterministic_HonestLedgerTracking(t *testing.T) {
	// Provider returned a non-JSON conversational refusal or malformed payload, but consumed tokens
	malformedResponse := "I apologize, but I cannot fulfill this request as formatted."

	mockClient := &mockLLMClient{
		response:     malformedResponse,
		inputTokens:  750,
		outputTokens: 50,
	}
	mockCosts := &mockCostRecorder{}
	cfg := sampleExplainerConfig()

	explainer := NewIncidentExplainer(mockClient, nil, mockCosts, cfg)
	anomaly := sampleAnomaly()

	res, err := explainer.Explain(context.Background(), "org-123", anomaly, false)
	if err != nil {
		t.Fatalf("unexpected error on malformed JSON: %v", err)
	}

	if !res.DegradedToDeterministic {
		t.Errorf("expected DegradedToDeterministic to be true on malformed JSON")
	}
	if !strings.Contains(res.FallbackReason, "did not adhere to required JSON schema") {
		t.Errorf("expected fallback reason to cite JSON schema failure, got: %s", res.FallbackReason)
	}

	// CRITICAL TEST: Honest ledger recording for consumed tokens
	if len(mockCosts.recorded) != 1 {
		t.Fatalf("expected 1 cost event recorded in ledger for consumed tokens, got %d", len(mockCosts.recorded))
	}
	if mockCosts.recorded[0].InputTokens != 750 || mockCosts.recorded[0].OutputTokens != 50 {
		t.Errorf("expected recorded tokens (750, 50), got (%d, %d)", mockCosts.recorded[0].InputTokens, mockCosts.recorded[0].OutputTokens)
	}

	expectedCost := (750.0*0.075 + 50.0*0.300) / 1_000_000.0
	if res.CostAttribution.EstimatedCostUSD != expectedCost {
		t.Errorf("expected cost attribution to reflect consumed tokens (%f), got %f", expectedCost, res.CostAttribution.EstimatedCostUSD)
	}
}

func TestGeminiClient_HTTPErrorParsing_WithMockServer(t *testing.T) {
	t.Run("Parses_Google_JSON_429_Rate_Limit_Error", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{
				"error": {
					"code": 429,
					"message": "Resource has been exhausted (e.g. check quota).",
					"status": "RESOURCE_EXHAUSTED"
				}
			}`))
		}))
		defer mockServer.Close()

		client := NewGeminiClient("test-key", "gemini-1.5-flash")
		client.baseURL = mockServer.URL
		client.httpClient = mockServer.Client()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, _, _, err := client.Generate(ctx, "sys", "usr")
		if err == nil {
			t.Fatalf("expected error from 429 mock server, got nil")
		}
		if !strings.Contains(err.Error(), "code 429") || !strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
			t.Errorf("expected error to contain code 429 and RESOURCE_EXHAUSTED, got: %v", err)
		}
	})

	t.Run("Parses_Non_JSON_502_Gateway_Error", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`<html><title>502 Bad Gateway</title><body>502 Bad Gateway</body></html>`))
		}))
		defer mockServer.Close()

		client := NewGeminiClient("test-key", "gemini-1.5-flash")
		client.baseURL = mockServer.URL
		client.httpClient = mockServer.Client()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, _, _, err := client.Generate(ctx, "sys", "usr")
		if err == nil {
			t.Fatalf("expected error from 502 gateway, got nil")
		}
		if !strings.Contains(err.Error(), "status 502 Bad Gateway") {
			t.Errorf("expected error to preserve status 502 Bad Gateway, got: %v", err)
		}
	})
}


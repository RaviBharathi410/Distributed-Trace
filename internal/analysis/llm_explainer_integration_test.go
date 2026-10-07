package analysis

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestGeminiClient_LiveIntegration exercises a live network round-trip against Google's Gemini API
// when GEMINI_API_KEY or LLM_API_KEY is present in the environment. If unconfigured, it skips cleanly.
func TestGeminiClient_LiveIntegration(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
	}
	if apiKey == "" {
		t.Skip("skipping live Gemini API integration test: GEMINI_API_KEY / LLM_API_KEY environment variable not set. Export key to run live integration test.")
	}

	client := NewGeminiClient(apiKey, "gemini-1.5-flash")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	userPrompt := BuildUserPrompt(sampleAnomaly())

	rawResp, inTokens, outTokens, err := client.Generate(ctx, systemPrompt, userPrompt)
	if err != nil {
		t.Fatalf("live Gemini Generate failed: %v", err)
	}

	if inTokens <= 0 {
		t.Errorf("expected positive input token count, got %d", inTokens)
	}
	if outTokens <= 0 {
		t.Errorf("expected positive output token count, got %d", outTokens)
	}

	var parsed llmOutputSchema
	if err := json.Unmarshal([]byte(rawResp), &parsed); err != nil {
		t.Fatalf("failed to unmarshal live response JSON (%q): %v", rawResp, err)
	}

	if parsed.Summary == "" {
		t.Errorf("expected non-empty summary from live Gemini API")
	}
	if len(parsed.ContributingFactors) == 0 {
		t.Errorf("expected at least 1 contributing factor from live Gemini API")
	}

	// Validate output against safety rules
	validator := NewOutputValidator()
	if err := validator.ValidateDiagnosticOutput(parsed.Summary, parsed.ContributingFactors); err != nil {
		t.Logf("Notice: live model generated output that triggered safety validator: %v", err)
	}
}

// TestIncidentExplainer_LiveEndToEnd_WithRealGemini tests the complete pipeline
// (prompt generation -> live API -> cost accounting -> caching) with live credentials.
func TestIncidentExplainer_LiveEndToEnd_WithRealGemini(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
	}
	if apiKey == "" {
		t.Skip("skipping live IncidentExplainer integration test: GEMINI_API_KEY / LLM_API_KEY environment variable not set.")
	}

	client := NewGeminiClient(apiKey, "gemini-1.5-flash")
	mockCosts := &mockCostRecorder{}
	cfg := sampleExplainerConfig()
	explainer := NewIncidentExplainer(client, NewOutputValidator(), mockCosts, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	anomaly := sampleAnomaly()
	explanation, err := explainer.Explain(ctx, "org-live-test", anomaly, false)
	if err != nil {
		t.Fatalf("unexpected error during live explain: %v", err)
	}

	if explanation.AnomalyID != anomaly.ID {
		t.Errorf("expected anomaly id %s, got %s", anomaly.ID, explanation.AnomalyID)
	}

	// Regardless of whether output was compliant or degraded to deterministic fallback,
	// tokens must have been consumed and recorded in the cost ledger
	if len(mockCosts.recorded) != 1 {
		t.Errorf("expected 1 cost event recorded in ledger, got %d", len(mockCosts.recorded))
	} else {
		rec := mockCosts.recorded[0]
		if rec.InputTokens <= 0 || rec.OutputTokens <= 0 {
			t.Errorf("expected real positive tokens recorded, got in=%d, out=%d", rec.InputTokens, rec.OutputTokens)
		}
		if rec.EstimatedCostUSD <= 0.0 {
			t.Errorf("expected real non-zero cost recorded, got %f", rec.EstimatedCostUSD)
		}
	}
}

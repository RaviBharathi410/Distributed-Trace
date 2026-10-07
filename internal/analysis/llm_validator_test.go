package analysis

import (
	"errors"
	"testing"
)

func TestOutputValidator_RejectsLiteralCommands(t *testing.T) {
	validator := NewOutputValidator()

	commandInputs := []string{
		"Run kubectl scale deployment payment-svc --replicas=5 to resolve the issue.",
		"Please perform a rollback of the previous commit.",
		"You must restart service payment-db immediately.",
		"Execute helm upgrade payment-chart to deploy the fix.",
		"Apply -f manifest.yaml to restore normal operations.",
		"Scale up the cluster node pool to absorb traffic spikes.",
	}

	for _, input := range commandInputs {
		err := validator.ValidateText(input)
		if err == nil {
			t.Errorf("expected validation error for command input: %q, got nil", input)
		}
		if !errors.Is(err, ErrRemediationDirectivesFound) {
			t.Errorf("expected ErrRemediationDirectivesFound, got: %v", err)
		}
	}
}

func TestOutputValidator_RejectsSoftAdvisoryPhrasing(t *testing.T) {
	validator := NewOutputValidator()

	advisoryInputs := []string{
		"This pattern is often resolved by increasing connection pool size on payment-svc.",
		"Typically indicates the service needs more replicas to handle current ingress.",
		"This is commonly addressed by a cache warm restart.",
		"The service needs scaling.",
		"We recommend increasing the database query timeout to 5000ms.",
		"You should configure connection pool max idle connections to 50.",
		"Consider tuning PostgreSQL shared buffers to mitigate buffer lock contention.",
		"The best practice is to increase replica count during peak hours.",
		"The service needs additional thread allocation to prevent queuing.",
		"Try restarting the pod to release stale file descriptors.",
	}

	for _, input := range advisoryInputs {
		err := validator.ValidateText(input)
		if err == nil {
			t.Errorf("expected validation error for soft advisory input: %q, got nil", input)
		}
		if !errors.Is(err, ErrSoftAdvisoryFound) {
			t.Errorf("expected ErrSoftAdvisoryFound, got: %v", err)
		}
	}
}

func TestOutputValidator_AcceptsPureDescriptiveTelemetry(t *testing.T) {
	validator := NewOutputValidator()

	validInputs := []struct {
		summary string
		factors []string
	}{
		{
			summary: "Database connection acquisition wait time in payment-svc spiked from 15ms to 240ms, causing request queue buildup.",
			factors: []string{
				"Active connections reached the configured pool limit of 100 at 14:02:10 UTC.",
				"Upstream edge checkout-api -> payment-svc observed 260ms p95 latency while error rate remained at 0.4%.",
				"Downstream PostgreSQL query duration remained flat at 3ms, isolating the delay to connection acquisition.",
			},
		},
		{
			summary: "Upstream gateway observed elevated HTTP 504 status codes due to downstream timeout in inventory-svc.",
			factors: []string{
				"P99 latency on GET /items reached 3200ms compared to baseline 45ms (z-score 6.2).",
				"Dependency graph critical path traversal identified lock contention on table inventory_locks.",
			},
		},
	}

	for _, vi := range validInputs {
		if err := validator.ValidateDiagnosticOutput(vi.summary, vi.factors); err != nil {
			t.Errorf("expected valid telemetry observation to pass, got error: %v", err)
		}
	}
}

func TestOutputValidator_TrueNegatives_ObservationalTelemetryWithTriggerWords(t *testing.T) {
	validator := NewOutputValidator()

	// These sentences contain trigger words like 'typically', 'often', 'needs', 'resolved', 'baseline'
	// but are purely observational telemetry statements, NOT remediation advice.
	observationalInputs := []string{
		"Latency typically rises during the 2–4pm window, consistent with the 7-day baseline.",
		"Error rate was often elevated during batch synchronizations, which resolved naturally once the queue drained.",
		"Database connection needs 12ms on average to establish handshake under normal network load.",
		"Historical anomaly was resolved automatically when background compaction completed at 03:00 UTC.",
		"The observed latency needs to be evaluated against the rolling Welford mean of 45ms.",
		"The trace waterfall indicates that upstream callers often retry failed idempotent GET requests.",
		"Cache hit ratio typically fluctuates between 85% and 95% across diurnal traffic cycles.",
		"The database connection pool acquisition latency was resolved when downstream lock contention subsided.",
	}

	for _, input := range observationalInputs {
		if err := validator.ValidateText(input); err != nil {
			t.Errorf("FALSE POSITIVE: valid observational telemetry was rejected by validator: %q, error: %v", input, err)
		}
	}
}


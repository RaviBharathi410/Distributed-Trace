package analysis

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrRemediationDirectivesFound = errors.New("output contains forbidden remediation or infrastructure mutation directives")
	ErrSoftAdvisoryFound          = errors.New("output contains forbidden soft advisory or prescriptive phrasing")
)

// Tier 1: Literal operational commands / tooling blocklist
var tier1CommandPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(kubectl|helm|docker|containerd|nerdctl|crictl)\b`),
	regexp.MustCompile(`(?i)\b(rollback|rollout|revert commit|git revert)\b`),
	regexp.MustCompile(`(?i)\b(scale up|scale down|autoscale|hpa)\b`),
	regexp.MustCompile(`(?i)\b(reboot|restart pod|restart service|kill -9|pkill)\b`),
	regexp.MustCompile(`(?i)\b(apply -f|terraform apply|ansible-playbook)\b`),
}

// Tier 2: Soft advisory / prescriptive modal regex patterns
var tier2AdvisoryPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b((can be|often|typically|commonly|usually|is best) (resolved|addressed|fixed|remedied|mitigated) by)\b`),
	regexp.MustCompile(`(?i)\b(pattern is (often |typically )?(resolved|addressed|fixed|mitigated) by)\b`),
	regexp.MustCompile(`(?i)\b(workaround is to)\b`),
	regexp.MustCompile(`(?i)\b(should (increase|decrease|add|scale|restart|reconfigure|tune|upgrade|revert|deploy|apply|modify))\b`),
	regexp.MustCompile(`(?i)\b(operator should|team should|user should|you should)\b`),
	regexp.MustCompile(`(?i)\b(needs to be (restarted|scaled|reconfigured|upgraded|reverted|increased|decreased|tuned|deployed))\b`),
	regexp.MustCompile(`(?i)\b(needs (more|fewer|additional) (replicas|memory|cpu|threads?|allocation|capacity|instances|nodes|connections))\b`),
	regexp.MustCompile(`(?i)\b(needs (scaling|tuning|restarting))\b`),
	regexp.MustCompile(`(?i)\b(recommend(ed)? (increasing|decreasing|adding|scaling|restarting|configuring|tuning|to))\b`),
	regexp.MustCompile(`(?i)\b(consider (increasing|decreasing|adding|scaling|restarting|tuning|adjusting|migrating))\b`),
	regexp.MustCompile(`(?i)\b(try (restarting|increasing|scaling|reverting|tuning))\b`),
	regexp.MustCompile(`(?i)\b(best practice is to|solution is to|next step is to)\b`),
	regexp.MustCompile(`(?i)\b(typically indicates the service needs)\b`),
}

// OutputValidator inspects LLM-generated diagnosis text to ensure strict adherence to Phase 5A
// read-only scope, preventing direct command injection and soft advisory phrasing.
type OutputValidator struct {
	tier1 []*regexp.Regexp
	tier2 []*regexp.Regexp
}

func NewOutputValidator() *OutputValidator {
	return &OutputValidator{
		tier1: tier1CommandPatterns,
		tier2: tier2AdvisoryPatterns,
	}
}

// ValidateText checks whether a text block violates Tier 1 (commands) or Tier 2 (soft advisory).
func (v *OutputValidator) ValidateText(text string) error {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}

	// Check Tier 1 (Literal Commands)
	for _, p := range v.tier1 {
		if loc := p.FindString(trimmed); loc != "" {
			return fmt.Errorf("%w: matched command pattern '%s'", ErrRemediationDirectivesFound, loc)
		}
	}

	// Check Tier 2 (Soft Advisory Phrases)
	for _, p := range v.tier2 {
		if loc := p.FindString(trimmed); loc != "" {
			return fmt.Errorf("%w: matched advisory pattern '%s'", ErrSoftAdvisoryFound, loc)
		}
	}

	return nil
}

// ValidateDiagnosticOutput validates all text fields of an incident explanation payload.
func (v *OutputValidator) ValidateDiagnosticOutput(summary string, contributingFactors []string) error {
	if err := v.ValidateText(summary); err != nil {
		return fmt.Errorf("summary violation: %w", err)
	}

	for i, factor := range contributingFactors {
		if err := v.ValidateText(factor); err != nil {
			return fmt.Errorf("contributing factor [%d] violation: %w", i, err)
		}
	}

	return nil
}

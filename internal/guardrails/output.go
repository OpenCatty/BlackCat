package guardrails

import "strings"

const maxOutputLength = 50000

// OutputGuardrail checks output quality rules.
type OutputGuardrail struct {
	enabled bool
}

// NewOutputGuardrail constructs an output guardrail.
func NewOutputGuardrail(enabled bool) *OutputGuardrail {
	return &OutputGuardrail{enabled: enabled}
}

// Check evaluates output quality.
func (g *OutputGuardrail) Check(output string) GuardrailResult {
	if g == nil || !g.enabled {
		return GuardrailResult{Allow: true}
	}

	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return GuardrailResult{Allow: false, Reason: "output is empty"}
	}

	if strings.HasPrefix(trimmed, "Error:") {
		return GuardrailResult{Allow: false, Reason: "output starts with raw error"}
	}

	if len(output) >= maxOutputLength {
		return GuardrailResult{Allow: false, Reason: "output exceeds length limit"}
	}

	return GuardrailResult{Allow: true}
}

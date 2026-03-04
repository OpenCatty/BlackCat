package guardrails

import "regexp"

var defaultInputPatterns = []string{
	`(?i)ignore\s+previous\s+instructions`,
	`(?i)system\s+prompt`,
	`(?i)jailbreak`,
	`(?i)forget\s+your\s+instructions`,
	`(?i)act\s+as`,
	`(?i)you\s+are\s+now`,
	`(?i)pretend\s+you\s+are`,
	`(?i)bypass`,
}

// InputGuardrail checks user input for prompt injection patterns.
type InputGuardrail struct {
	enabled  bool
	patterns []*regexp.Regexp
}

// NewInputGuardrail constructs an input guardrail.
func NewInputGuardrail(enabled bool, customPatterns []string) *InputGuardrail {
	patterns := make([]*regexp.Regexp, 0, len(defaultInputPatterns)+len(customPatterns))

	for _, pattern := range defaultInputPatterns {
		re, err := regexp.Compile(pattern)
		if err == nil {
			patterns = append(patterns, re)
		}
	}

	for _, pattern := range customPatterns {
		re, err := regexp.Compile(pattern)
		if err == nil {
			patterns = append(patterns, re)
		}
	}

	return &InputGuardrail{enabled: enabled, patterns: patterns}
}

// Check evaluates input text.
func (g *InputGuardrail) Check(input string) GuardrailResult {
	if g == nil || !g.enabled {
		return GuardrailResult{Allow: true}
	}

	for _, re := range g.patterns {
		if re.MatchString(input) {
			return GuardrailResult{Allow: false, Reason: "input blocked by prompt injection guardrail"}
		}
	}

	return GuardrailResult{Allow: true}
}

package guardrails

import (
	"regexp"
	"strings"
)

var defaultToolPatterns = []string{
	`(?i)\brm\s+-rf\b`,
	`(?i)\brm\s+-r\s+/\b`,
	`(?i)\bgit\s+push\s+--force\b`,
	`(?i)\bgit\s+push\s+-f\b`,
	`(?i)\bDROP\s+TABLE\b`,
	`(?i)\bDROP\s+DATABASE\b`,
	`(?i)\bformat\s+c:\b`,
	`(?i)\bmkfs\b`,
	`(?i)\bdd\s+if=`,
}

// ToolGuardrail checks tool commands for dangerous patterns.
type ToolGuardrail struct {
	enabled  bool
	patterns []*regexp.Regexp
}

// NewToolGuardrail constructs a tool guardrail.
func NewToolGuardrail(enabled bool, requireApprovalPatterns []string) *ToolGuardrail {
	patterns := make([]*regexp.Regexp, 0, len(defaultToolPatterns)+len(requireApprovalPatterns))

	for _, pattern := range defaultToolPatterns {
		re, err := regexp.Compile(pattern)
		if err == nil {
			patterns = append(patterns, re)
		}
	}

	for _, pattern := range requireApprovalPatterns {
		re, err := regexp.Compile(pattern)
		if err == nil {
			patterns = append(patterns, re)
		}
	}

	return &ToolGuardrail{enabled: enabled, patterns: patterns}
}

// Check evaluates tool name and arguments.
func (g *ToolGuardrail) Check(toolName, toolArgs string) GuardrailResult {
	if g == nil || !g.enabled {
		return GuardrailResult{Allow: true}
	}

	payload := strings.TrimSpace(toolName + " " + toolArgs)
	for _, re := range g.patterns {
		if re.MatchString(payload) {
			return GuardrailResult{Allow: false, Reason: "tool blocked by dangerous command guardrail"}
		}
	}

	return GuardrailResult{Allow: true}
}

package security

import (
	"regexp"
	"strings"
)

// scrubRule holds a compiled regexp and its replacement logic.
type scrubRule struct {
	re          *regexp.Regexp
	replacement string
	// replaceFunc is used when simple replacement isn't sufficient.
	replaceFunc func(string) string
}

// Scrubber detects and redacts credentials in text.
type Scrubber struct {
	rules []scrubRule
}

// NewScrubber compiles all credential detection patterns and returns a Scrubber.
func NewScrubber() *Scrubber {
	s := &Scrubber{}

	// OpenAI keys: sk-<20+ alphanumeric>
	s.addSimple(`sk-[a-zA-Z0-9]{20,}`, "[REDACTED]")

	// GitHub tokens: ghp_ and gho_
	s.addSimple(`ghp_[a-zA-Z0-9]{36}`, "[REDACTED]")
	s.addSimple(`gho_[a-zA-Z0-9]{36}`, "[REDACTED]")

	// Slack tokens: xoxb- and xoxp-
	s.addSimple(`xoxb-[a-zA-Z0-9-]+`, "[REDACTED]")
	s.addSimple(`xoxp-[a-zA-Z0-9-]+`, "[REDACTED]")

	// AWS access keys: AKIA followed by 16 uppercase alphanumeric
	s.addSimple(`AKIA[0-9A-Z]{16}`, "[REDACTED]")

	// Passwords in URLs: ://user:password@host
	// Capture groups: $1=scheme+user:, $2=password, $3=@
	// We redact only the password part.
	s.addFunc(`(://[^:]+:)([^@]+)(@)`, func(match string) string {
		re := regexp.MustCompile(`(://[^:]+:)([^@]+)(@)`)
		return re.ReplaceAllString(match, "${1}[REDACTED]${3}")
	})

	// Generic API key patterns: api_key=value, secret=value, token=value, password=value
	// Case insensitive. Value must be 16+ chars.
	s.addFunc(`(?i)((?:api[_-]?key|secret|token|password)\s*[:=]\s*["']?)([a-zA-Z0-9_\-./+=]{16,})`, func(match string) string {
		re := regexp.MustCompile(`(?i)((?:api[_-]?key|secret|token|password)\s*[:=]\s*["']?)([a-zA-Z0-9_\-./+=]{16,})`)
		return re.ReplaceAllString(match, "${1}[REDACTED]")
	})

	// AWS secret keys: 40-char base64-ish string near "aws" or "secret" context
	// We handle this via a context-aware rule in Scrub method.
	// Not added as a simple rule to avoid false positives.

	return s
}

// addSimple adds a rule that replaces the entire match with the given string.
func (s *Scrubber) addSimple(pattern, replacement string) {
	re := regexp.MustCompile(pattern)
	s.rules = append(s.rules, scrubRule{re: re, replacement: replacement})
}

// addFunc adds a rule that uses a function for replacement.
func (s *Scrubber) addFunc(pattern string, fn func(string) string) {
	re := regexp.MustCompile(pattern)
	s.rules = append(s.rules, scrubRule{re: re, replaceFunc: fn})
}

// awsSecretKeyRe matches 40-char base64-ish strings that could be AWS secret keys.
var awsSecretKeyRe = regexp.MustCompile(`[a-zA-Z0-9/+=]{40}`)

// Scrub replaces all detected credentials in text with [REDACTED].
func (s *Scrubber) Scrub(text string) string {
	result := text

	for _, rule := range s.rules {
		if rule.replaceFunc != nil {
			result = rule.re.ReplaceAllStringFunc(result, rule.replaceFunc)
		} else {
			result = rule.re.ReplaceAllString(result, rule.replacement)
		}
	}

	// AWS secret key: context-aware — only scrub 40-char strings near "aws" or "secret"
	lower := strings.ToLower(result)
	if strings.Contains(lower, "aws") || strings.Contains(lower, "secret") {
		result = awsSecretKeyRe.ReplaceAllStringFunc(result, func(match string) string {
			// Check if this match is near "aws" or "secret" in the original lowered text
			idx := strings.Index(lower, strings.ToLower(match))
			if idx < 0 {
				return match
			}
			// Look in a window around the match for context keywords
			start := idx - 100
			if start < 0 {
				start = 0
			}
			end := idx + len(match) + 100
			if end > len(lower) {
				end = len(lower)
			}
			window := lower[start:end]
			if strings.Contains(window, "aws") || strings.Contains(window, "secret") {
				return "[REDACTED]"
			}
			return match
		})
	}

	return result
}

// ScrubAll scrubs multiple strings, returning a new slice with all credentials redacted.
func (s *Scrubber) ScrubAll(texts []string) []string {
	out := make([]string, len(texts))
	for i, t := range texts {
		out[i] = s.Scrub(t)
	}
	return out
}

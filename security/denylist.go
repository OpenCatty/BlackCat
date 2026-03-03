// Package security provides shell command deny-listing and credential scrubbing.
package security

import (
	"errors"
	"fmt"
	"regexp"
)

// ErrDenyListViolation is the sentinel error for blocked commands.
var ErrDenyListViolation = errors.New("command blocked by deny list")

// DenyListViolation wraps ErrDenyListViolation with the matched pattern and command.
type DenyListViolation struct {
	Pattern string
	Command string
}

func (v *DenyListViolation) Error() string {
	return fmt.Sprintf("command blocked by deny list: pattern %q matched command %q", v.Pattern, v.Command)
}

func (v *DenyListViolation) Unwrap() error {
	return ErrDenyListViolation
}

// denyEntry holds a compiled regexp and its source pattern string.
type denyEntry struct {
	re      *regexp.Regexp
	pattern string
}

// defaultPatterns are the built-in deny-list regex patterns.
// These use RE2 syntax (no backreferences, no lookahead).
var defaultPatterns = []string{
	// Pipe curl/wget output to shell
	`curl\s+.*\|\s*(ba)?sh`,
	`wget\s+.*\|\s*(ba)?sh`,

	// bash -c execution (but "bash" alone is fine)
	`bash\s+-c\s+`,

	// eval subshell
	`eval\s+\$\(`,

	// base64 decode piped to shell
	`base64\s+.*\|\s*(ba)?sh`,

	// Bash reverse shell via /dev/tcp
	`/dev/tcp/`,

	// Netcat reverse shell
	`nc\s+.*-e`,

	// Named pipe (reverse shell component)
	`mkfifo\s+`,

	// rm -rf / (but allow rm -rf /tmp/foo)
	// Match "rm -rf /" at end of string OR "rm -rf /" followed by non-letter
	`rm\s+-rf\s+/\s*$`,
	`rm\s+-rf\s+/[^a-zA-Z]`,

	// Disk overwrite
	`dd\s+if=`,

	// chmod 777 on root
	`chmod\s+777\s+/`,

	// Fork bomb: :(){:|:&};:
	`:\(\)\{\s*:\|:&\s*\};:`,
}

// DenyList checks shell commands against a set of compiled regex patterns.
type DenyList struct {
	entries []denyEntry
}

// NewDenyList compiles the default deny patterns plus any extra patterns provided.
// Panics if any pattern fails to compile (programming error).
func NewDenyList(extraPatterns ...string) *DenyList {
	all := make([]string, 0, len(defaultPatterns)+len(extraPatterns))
	all = append(all, defaultPatterns...)
	all = append(all, extraPatterns...)

	entries := make([]denyEntry, 0, len(all))
	for _, p := range all {
		re, err := regexp.Compile(p)
		if err != nil {
			panic(fmt.Sprintf("security: failed to compile deny pattern %q: %v", p, err))
		}
		entries = append(entries, denyEntry{re: re, pattern: p})
	}

	return &DenyList{entries: entries}
}

// Check returns nil if the command is safe, or a *DenyListViolation if it
// matches any deny pattern. The returned error wraps ErrDenyListViolation.
func (d *DenyList) Check(command string) error {
	for _, e := range d.entries {
		if e.re.MatchString(command) {
			return &DenyListViolation{
				Pattern: e.pattern,
				Command: command,
			}
		}
	}
	return nil
}

package rules

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Rule represents a conditional rule loaded from a .md file.
type Rule struct {
	Name     string
	Globs    []string // glob patterns from YAML frontmatter
	Content  string   // markdown body (rule content to inject)
	FilePath string   // source .md file path
}

// Engine loads and matches rules.
type Engine struct {
	rules []Rule
}

// NewEngine creates a new rule engine.
func NewEngine() *Engine {
	return &Engine{}
}

// ruleFrontmatter represents YAML frontmatter in a rule .md file.
type ruleFrontmatter struct {
	Name  string   `yaml:"name"`
	Globs []string `yaml:"globs"`
}

// LoadRules reads all .md files from dir, parses YAML frontmatter for globs field,
// and stores rules in memory. Files without frontmatter or without globs are skipped with warning.
func (e *Engine) LoadRules(dir string) error {
	// If dir doesn't exist, return empty (not error)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat rules dir: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("rules path is not a directory: %s", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read rules dir: %w", err)
	}

	var rules []Rule

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Warning: cannot read rule file %s: %v", filePath, err)
			continue
		}

		rule, ok := parseRuleFile(string(content), filePath)
		if !ok {
			continue // warnings already logged in parseRuleFile
		}

		rules = append(rules, rule)
	}

	e.rules = rules
	return nil
}

// parseRuleFile parses a rule .md file with YAML frontmatter.
// Returns (rule, true) on success, or (zero, false) if file should be skipped.
func parseRuleFile(content string, filePath string) (Rule, bool) {
	// Check for frontmatter delimiter
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		log.Printf("Warning: rule file %s has no frontmatter, skipping", filePath)
		return Rule{}, false
	}

	// Find start offset after opening ---
	var startIdx int
	if strings.HasPrefix(content, "---\r\n") {
		startIdx = 5
	} else {
		startIdx = 4
	}

	// Find closing delimiter
	remainder := content[startIdx:]
	lines := strings.Split(remainder, "\n")

	var yamlContent strings.Builder
	endIdx := -1

	for i, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		if trimmed == "---" || trimmed == "..." {
			endIdx = i
			break
		}
		if i > 0 {
			yamlContent.WriteString("\n")
		}
		yamlContent.WriteString(line)
	}

	if endIdx == -1 {
		log.Printf("Warning: rule file %s has unclosed frontmatter, skipping", filePath)
		return Rule{}, false
	}

	// Parse YAML
	var fm ruleFrontmatter
	if err := yaml.Unmarshal([]byte(yamlContent.String()), &fm); err != nil {
		log.Printf("Warning: rule file %s has invalid YAML frontmatter: %v, skipping", filePath, err)
		return Rule{}, false
	}

	// Must have globs
	if len(fm.Globs) == 0 {
		log.Printf("Warning: rule file %s has no globs in frontmatter, skipping", filePath)
		return Rule{}, false
	}

	// Validate globs, warn and filter invalid ones
	var validGlobs []string
	for _, g := range fm.Globs {
		if _, err := path.Match(g, "test"); err != nil {
			log.Printf("Warning: rule file %s has invalid glob pattern %q: %v, skipping pattern", filePath, g, err)
			continue
		}
		validGlobs = append(validGlobs, g)
	}

	if len(validGlobs) == 0 {
		log.Printf("Warning: rule file %s has no valid glob patterns, skipping", filePath)
		return Rule{}, false
	}

	// Extract body content
	bodyLines := lines[endIdx+1:]
	body := strings.Join(bodyLines, "\n")
	body = strings.TrimSpace(body)

	// Use frontmatter name or derive from filename
	name := fm.Name
	if name == "" {
		base := filepath.Base(filePath)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	return Rule{
		Name:     name,
		Globs:    validGlobs,
		Content:  body,
		FilePath: filePath,
	}, true
}

// Match returns all rules whose glob patterns match the given filePath.
// E9: normalizes filePath with filepath.ToSlash() before matching.
func (e *Engine) Match(filePath string) []Rule {
	// E9: normalize to forward slashes for cross-platform glob matching
	normalized := filepath.ToSlash(filePath)

	var matched []Rule

	for _, rule := range e.rules {
		if ruleMatches(rule, normalized) {
			matched = append(matched, rule)
		}
	}

	return matched
}

// ruleMatches checks if any of the rule's glob patterns match the normalized path.
func ruleMatches(rule Rule, normalizedPath string) bool {
	for _, pattern := range rule.Globs {
		// Normalize pattern to forward slashes too
		pattern = filepath.ToSlash(pattern)

		if globMatch(pattern, normalizedPath) {
			return true
		}
	}
	return false
}

// globMatch performs glob matching that supports ** for recursive directory matching.
// path.Match does NOT support **, so we handle it:
// - Split pattern on ** segments
// - For single-segment patterns (no **), use path.Match directly
// - For ** patterns, match recursively
func globMatch(pattern, name string) bool {
	// If pattern contains no **, use path.Match directly
	if !strings.Contains(pattern, "**") {
		matched, err := path.Match(pattern, name)
		if err != nil {
			return false
		}
		return matched
	}

	// Handle ** patterns by splitting on "**/" or "/**/" or "/**"
	// Strategy: ** matches zero or more path segments
	return matchDoublestar(pattern, name)
}

// matchDoublestar handles ** glob patterns.
// ** matches zero or more directory segments (including none).
func matchDoublestar(pattern, filePath string) bool {
	// Split pattern into parts around **
	parts := strings.Split(pattern, "**")

	if len(parts) == 2 {
		prefix := parts[0]
		suffix := parts[1]

		// Remove trailing/leading slashes from prefix/suffix for cleaner matching
		if strings.HasSuffix(prefix, "/") {
			prefix = prefix[:len(prefix)-1]
		}
		if strings.HasPrefix(suffix, "/") {
			suffix = suffix[1:]
		}

		// If prefix is empty, ** is at the start: match any path ending with suffix
		if prefix == "" {
			// Match suffix against filename or path tail
			if suffix == "" {
				return true // "**" matches everything
			}
			// Try matching suffix against path and all subpaths
			pathParts := strings.Split(filePath, "/")
			for i := 0; i < len(pathParts); i++ {
				subpath := strings.Join(pathParts[i:], "/")
				matched, err := path.Match(suffix, subpath)
				if err == nil && matched {
					return true
				}
			}
			return false
		}

		// If suffix is empty, ** is at the end: match any path starting with prefix
		if suffix == "" {
			matched, err := path.Match(prefix, filePath)
			if err == nil && matched {
				return true
			}
			// Also match if path starts with prefix/
			if strings.HasPrefix(filePath, prefix+"/") {
				return true
			}
			return false
		}

		// Both prefix and suffix exist: prefix/**/suffix
		// Path must start with something matching prefix and end with something matching suffix
		pathParts := strings.Split(filePath, "/")
		for i := 0; i < len(pathParts); i++ {
			prefixCandidate := strings.Join(pathParts[:i+1], "/")
			prefixMatched, err := path.Match(prefix, prefixCandidate)
			if err != nil {
				continue
			}
			if !prefixMatched {
				continue
			}

			for j := i; j < len(pathParts); j++ {
				suffixCandidate := strings.Join(pathParts[j:], "/")
				suffixMatched, err2 := path.Match(suffix, suffixCandidate)
				if err2 != nil {
					continue
				}
				if suffixMatched {
					return true
				}
			}
		}
		return false
	}

	// Multiple ** segments — fall back to trying all combinations
	// This is a rare edge case; for simplicity, return false
	return false
}

// Rules returns a copy of all loaded rules.
func (e *Engine) Rules() []Rule {
	result := make([]Rule, len(e.rules))
	copy(result, e.rules)
	return result
}

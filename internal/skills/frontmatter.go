package skills

import (
	"log"
	"strings"

	"gopkg.in/yaml.v3"
)

// MCPConfig represents a Model Context Protocol server configuration
type MCPConfig struct {
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args"`
	Env     map[string]string `yaml:"env"`
}

// Requirements represents required binaries and environment variables
type Requirements struct {
	Bins    []string   `yaml:"bins"`     // required binaries on PATH
	Env     []string   `yaml:"env"`      // required environment variables
	AnyBins [][]string `yaml:"any_bins"` // each inner slice is an OR group; all groups are AND-ed
}

// FrontmatterData represents YAML frontmatter metadata in a skill file
type FrontmatterData struct {
	Name        string               `yaml:"name"`
	Description string               `yaml:"description"`
	MCP         map[string]MCPConfig `yaml:"mcp"`
	Tags        []string             `yaml:"tags"`
	Requires    Requirements         `yaml:"requires"`
	Version     string               `yaml:"version"`
	Install     string               `yaml:"install"`
	DependsOn   []string             `yaml:"depends_on"`
}

// ParseFrontmatter parses YAML frontmatter from skill content.
// Returns (data, bodyContent, hasFrontmatter).
// If content does NOT start with "---\n", returns (zero value, content, false).
// If it starts with "---\n" but YAML is invalid: log warning, return (zero value, content, false).
// Detection logic:
// 1. If content starts with `---\n` → parse YAML between first `---\n` and second `---\n` (or `...\n`)
// 2. Body is everything after the closing delimiter
// 3. If no frontmatter → return zero FrontmatterData, original content, false
func ParseFrontmatter(content string) (FrontmatterData, string, bool) {
	// Check if content starts with frontmatter delimiter
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return FrontmatterData{}, content, false
	}

	// Find the closing delimiter (either --- or ...)
	var closeDelim string
	var startIdx int
	if strings.HasPrefix(content, "---\r\n") {
		startIdx = 5 // Length of "---\r\n"
		closeDelim = "---"
	} else {
		startIdx = 4 // Length of "---\n"
		closeDelim = "---"
	}

	// Look for closing delimiter on its own line
	remainder := content[startIdx:]
	lines := strings.Split(remainder, "\n")

	var yamlContent strings.Builder
	var endIdx int = -1

	for i, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		if trimmed == closeDelim || trimmed == "..." {
			endIdx = i
			break
		}
		if i > 0 {
			yamlContent.WriteString("\n")
		}
		yamlContent.WriteString(line)
	}

	// If no closing delimiter found, no valid frontmatter
	if endIdx == -1 {
		return FrontmatterData{}, content, false
	}

	// Parse YAML
	var fm FrontmatterData
	if err := yaml.Unmarshal([]byte(yamlContent.String()), &fm); err != nil {
		log.Printf("Warning: invalid YAML frontmatter, treating as regular skill: %v", err)
		return FrontmatterData{}, content, false
	}

	// Extract body content (everything after closing delimiter)
	bodyLines := lines[endIdx+1:]
	bodyContent := strings.Join(bodyLines, "\n")
	bodyContent = strings.TrimSpace(bodyContent)

	return fm, bodyContent, true
}

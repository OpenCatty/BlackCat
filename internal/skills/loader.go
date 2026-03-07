package skills

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

// Skill represents a skill loaded from a markdown file
type Skill struct {
	Name     string
	Content  string
	Tags     []string
	FilePath string
	Requires Requirements

	// Phase 3 fields
	Version   string     `json:"version,omitempty"`
	Install   string     `json:"-"` // intentionally excluded from serialization
	DependsOn []string   `json:"depends_on,omitempty"`
	AnyBins   [][]string `json:"-"` // runtime only
}

// ValidateVersion checks that v is a valid semantic version (e.g. "v1.2.3").
// Returns the canonical form or an error.
func ValidateVersion(v string) (string, error) {
	if v == "" {
		return "", nil
	}
	canonical := semver.Canonical(v)
	if canonical == "" {
		return "", fmt.Errorf("invalid semver %q: must be in vMAJOR.MINOR.PATCH form", v)
	}
	return canonical, nil
}

// IsEligible checks if a skill's requirements are met.
// Returns true if all required binaries are on PATH and all required env vars are set.
func (s *Skill) IsEligible() bool {
	for _, bin := range s.Requires.Bins {
		if _, err := exec.LookPath(bin); err != nil {
			return false
		}
	}
	for _, env := range s.Requires.Env {
		if os.Getenv(env) == "" {
			return false
		}
	}
	if !checkAnyBins(s.AnyBins) {
		return false
	}
	return true
}

// checkAnyBins returns true if every group in anyBins has at least one binary found on PATH.
// Each inner slice is an OR-group: at least one binary from the group must exist.
// Groups are AND-ed: ALL groups must be satisfied.
// Empty anyBins (nil or length 0) returns true.
func checkAnyBins(anyBins [][]string) bool {
	for _, group := range anyBins {
		found := false
		for _, bin := range group {
			if _, err := exec.LookPath(bin); err == nil {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// getBinaryVersion tries to get the version string of a binary by running common version flags.
// Returns the first semver-like string found, or empty string if not found or binary not on PATH.
// Uses a 3-second timeout.
func getBinaryVersion(bin string) string {
	if _, err := exec.LookPath(bin); err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for _, flag := range []string{"--version", "-v", "version"} {
		out, err := exec.CommandContext(ctx, bin, flag).CombinedOutput()
		if err != nil {
			continue
		}
		// Look for semver-like pattern (vX.Y.Z or X.Y.Z)
		re := regexp.MustCompile(`v?\d+\.\d+\.\d+`)
		if m := re.FindString(string(out)); m != "" {
			return m
		}
	}
	return ""
}

// LoadSkills loads all .md files from a directory and parses them as skills
func LoadSkills(dir string) ([]Skill, error) {
	// If dir doesn't exist, return empty slice (not error)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Skill{}, nil
		}
		return nil, err
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var skills []Skill

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Filter for .md files
		if filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue // Skip files that can't be read
		}

		skill := parseSkillFile(string(content), filePath, entry.Name())
		skills = append(skills, skill)
	}

	// Sort by Name (alphabetical)
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills, nil
}

// parseSkillFile parses a skill markdown file
func parseSkillFile(content string, filePath string, filename string) Skill {
	skill := Skill{
		FilePath: filePath,
	}

	// Try to parse YAML frontmatter first
	fm, body, hasFrontmatter := ParseFrontmatter(content)

	if hasFrontmatter {
		// If frontmatter present, use it for metadata
		if fm.Name != "" {
			skill.Name = fm.Name
		}
		if fm.Description != "" {
			// Description in frontmatter is optional metadata
		}
		if len(fm.Tags) > 0 {
			skill.Tags = fm.Tags
		}
		skill.Requires = fm.Requires
		skill.Version = fm.Version
		skill.Install = fm.Install
		skill.DependsOn = fm.DependsOn
		skill.AnyBins = fm.Requires.AnyBins
		skill.Content = body

		// If name not set from frontmatter, fall through to header parsing
		if skill.Name == "" {
			// Parse name from body content if not in frontmatter
			lines := strings.Split(body, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "# ") {
					skill.Name = strings.TrimPrefix(line, "# ")
					skill.Name = strings.TrimSpace(skill.Name)
					break
				}
			}
		}

		// If still no name, use filename
		if skill.Name == "" {
			skill.Name = strings.TrimSuffix(filename, filepath.Ext(filename))
		}
		return skill
	}

	// Fall back to original parsing logic for non-frontmatter skills (G3)
	// This ensures perfect backward compatibility
	lines := strings.Split(content, "\n")

	// Extract name from first # heading line, or use filename
	nameFound := false
	contentStartIdx := 0

	for i, line := range lines {
		if strings.HasPrefix(line, "# ") {
			skill.Name = strings.TrimPrefix(line, "# ")
			skill.Name = strings.TrimSpace(skill.Name)
			nameFound = true
			contentStartIdx = i + 1
			break
		}
	}

	// If no heading found, use filename without extension
	if !nameFound {
		skill.Name = strings.TrimSuffix(filename, filepath.Ext(filename))
	}

	// Parse tags from "Tags:" or "**Tags**:" line
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Tags:") || strings.HasPrefix(trimmed, "**Tags**:") {
			// Extract tags after "Tags:" or "**Tags**:"
			var tagsStr string
			if strings.HasPrefix(trimmed, "**Tags**:") {
				tagsStr = strings.TrimPrefix(trimmed, "**Tags**:")
			} else {
				tagsStr = strings.TrimPrefix(trimmed, "Tags:")
			}
			tagsStr = strings.TrimSpace(tagsStr)

			// Parse comma-separated tags
			if tagsStr != "" {
				tagParts := strings.Split(tagsStr, ",")
				for _, tag := range tagParts {
					tag := strings.TrimSpace(tag)
					if tag != "" {
						skill.Tags = append(skill.Tags, tag)
					}
				}
			}
			break
		}
	}

	// Content is everything after the first heading line
	if contentStartIdx > 0 && contentStartIdx < len(lines) {
		skill.Content = strings.Join(lines[contentStartIdx:], "\n")
		skill.Content = strings.TrimSpace(skill.Content)
	} else if contentStartIdx == 0 && nameFound {
		// If we found a name but contentStartIdx is 0, take everything after first heading
		if len(lines) > 1 {
			skill.Content = strings.Join(lines[1:], "\n")
			skill.Content = strings.TrimSpace(skill.Content)
		}
	} else {
		// No heading found, use all content
		skill.Content = strings.TrimSpace(content)
	}

	return skill
}

// LoadSkillsFromMultipleSources loads skills from multiple directories with precedence.
// Earlier directories have higher priority — if the same skill name exists in multiple
// directories, only the first occurrence is kept.
// Returns eligible skills only (those with satisfied requirements).
func LoadSkillsFromMultipleSources(dirs []string) ([]Skill, error) {
	seen := make(map[string]bool)
	var all []Skill
	for _, dir := range dirs {
		skills, err := LoadSkills(dir)
		if err != nil {
			continue // skip missing directories
		}
		for _, s := range skills {
			if !seen[s.Name] && s.IsEligible() {
				seen[s.Name] = true
				all = append(all, s)
			}
		}
	}
	sorted, err := ResolveDependencies(all)
	if err != nil {
		log.Printf("skills: dependency resolution warning: %v; loading without dependency order", err)
		return all, nil
	}
	return sorted, nil
}

// FormatForInjection formats skills as context blocks for system prompt injection
func FormatForInjection(skills []Skill) string {
	if len(skills) == 0 {
		return ""
	}

	var output strings.Builder

	for i, skill := range skills {
		output.WriteString(fmt.Sprintf("<skill name=\"%s\">\n", skill.Name))
		output.WriteString(skill.Content)
		output.WriteString("\n</skill>")

		// Add newline between skills, but not after the last one
		if i < len(skills)-1 {
			output.WriteString("\n\n")
		}
	}

	return output.String()
}

// FilterByFileSize returns skills whose raw content size is at or below maxBytes.
// Skills exceeding the limit are silently skipped.
func FilterByFileSize(skills []Skill, maxBytes int) []Skill {
	if maxBytes <= 0 {
		return skills
	}
	out := make([]Skill, 0, len(skills))
	for _, s := range skills {
		if len(s.Content) <= maxBytes {
			out = append(out, s)
		}
	}
	return out
}

// LimitSkillCount returns at most maxCount skills from the provided slice.
// If maxCount is 0 or negative, all skills are returned unchanged.
func LimitSkillCount(skills []Skill, maxCount int) []Skill {
	if maxCount <= 0 || len(skills) <= maxCount {
		return skills
	}
	return skills[:maxCount]
}

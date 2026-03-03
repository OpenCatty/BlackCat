package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Profile struct {
	Name                string
	SystemPromptOverlay string
	Tags                []string
	FilePath            string
}

type ProfileLoader struct{}

func (l *ProfileLoader) Load(dir string) (map[string]*Profile, error) {
	profiles := make(map[string]*Profile)

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return profiles, nil
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
			continue
		}

		profile := parseProfileFile(string(content), filePath, entry.Name())
		profiles[strings.ToLower(profile.Name)] = profile
	}

	return profiles, nil
}

func parseProfileFile(content string, filePath string, filename string) *Profile {
	profile := &Profile{FilePath: filePath}
	lines := strings.Split(content, "\n")

	headingIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			profile.Name = strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			headingIdx = i
			break
		}
	}

	if profile.Name == "" {
		profile.Name = strings.TrimSuffix(filename, filepath.Ext(filename))
	}

	start := 0
	if headingIdx >= 0 {
		start = headingIdx + 1
	}

	var overlayLines []string
	for _, line := range lines[start:] {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "tags:") {
			tagsPart := strings.TrimSpace(trimmed[len("tags:"):])
			if tagsPart != "" {
				tagParts := strings.Split(tagsPart, ",")
				for _, tag := range tagParts {
					tag = strings.TrimSpace(tag)
					if tag != "" {
						profile.Tags = append(profile.Tags, tag)
					}
				}
			}
			continue
		}

		overlayLines = append(overlayLines, line)
	}

	profile.SystemPromptOverlay = strings.TrimSpace(strings.Join(overlayLines, "\n"))
	return profile
}

func ApplyProfile(basePrompt string, profile *Profile) string {
	if profile == nil || strings.TrimSpace(profile.SystemPromptOverlay) == "" {
		return basePrompt
	}

	if strings.TrimSpace(basePrompt) == "" {
		return profile.SystemPromptOverlay
	}

	return profile.SystemPromptOverlay + "\n\n---\n\n" + basePrompt
}

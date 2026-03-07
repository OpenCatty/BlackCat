package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// RegistryEntry represents a single skill in the marketplace registry.
type RegistryEntry struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Install     string `json:"install"`
	Author      string `json:"author"`
	License     string `json:"license"`
}

// Registry represents the local marketplace registry file.
type Registry struct {
	Skills []RegistryEntry `json:"skills"`
}

// LoadRegistry reads and parses a registry.json file from marketplaceDir.
// Returns empty Registry (not error) if file does not exist.
// Returns error only on malformed JSON or unreadable file (non-NotExist errors).
func LoadRegistry(marketplaceDir string) (*Registry, error) {
	path := filepath.Join(marketplaceDir, "registry.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Registry{}, nil
		}
		return nil, err
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parsing registry.json: %w", err)
	}
	return &reg, nil
}

// InstalledStatus cross-references registry entries against loaded skills.
// Returns a map of skill-name → "installed" | "update-available" | "not-installed".
// "update-available" only if BOTH registry entry version AND loaded skill version are non-empty AND differ.
// "installed" if both versions match OR either version is empty.
func (r *Registry) InstalledStatus(loaded []Skill) map[string]string {
	loadedMap := make(map[string]string, len(loaded))
	for _, s := range loaded {
		loadedMap[s.Name] = s.Version
	}

	status := make(map[string]string, len(r.Skills))
	for _, entry := range r.Skills {
		loadedVersion, exists := loadedMap[entry.Name]
		if !exists {
			status[entry.Name] = "not-installed"
			continue
		}

		// Normalize both versions
		regVer, _ := ValidateVersion(entry.Version)
		loadVer, _ := ValidateVersion(loadedVersion)

		if regVer != "" && loadVer != "" && regVer != loadVer {
			status[entry.Name] = "update-available"
		} else {
			status[entry.Name] = "installed"
		}
	}
	return status
}

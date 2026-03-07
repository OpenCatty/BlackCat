package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadRegistry_FileNotExists verifies that a non-existent directory returns empty registry, no error.
func TestLoadRegistry_FileNotExists(t *testing.T) {
	r, err := LoadRegistry("/nonexistent/path/xyz")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(r.Skills) != 0 {
		t.Fatalf("expected 0 skills, got %d", len(r.Skills))
	}
}

// TestLoadRegistry_ValidJSON verifies that a valid registry.json is parsed correctly.
func TestLoadRegistry_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	data := `{"skills":[{"name":"test-skill","version":"v1.0.0","description":"A test","install":"copy it","author":"tester","license":"MIT"}]}`
	if err := os.WriteFile(filepath.Join(dir, "registry.json"), []byte(data), 0644); err != nil {
		t.Fatalf("failed to write registry.json: %v", err)
	}

	r, err := LoadRegistry(dir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(r.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(r.Skills))
	}
	if r.Skills[0].Name != "test-skill" {
		t.Fatalf("expected name %q, got %q", "test-skill", r.Skills[0].Name)
	}
	if r.Skills[0].Version != "v1.0.0" {
		t.Fatalf("expected version %q, got %q", "v1.0.0", r.Skills[0].Version)
	}
}

// TestLoadRegistry_MalformedJSON verifies that malformed JSON returns an error containing "parsing".
func TestLoadRegistry_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "registry.json"), []byte(`{not valid json`), 0644); err != nil {
		t.Fatalf("failed to write registry.json: %v", err)
	}

	_, err := LoadRegistry(dir)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
	if !strings.Contains(err.Error(), "parsing") {
		t.Fatalf("expected error to contain 'parsing', got: %v", err)
	}
}

// TestLoadRegistry_EmptySkills verifies that an empty skills array parses without error.
func TestLoadRegistry_EmptySkills(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "registry.json"), []byte(`{"skills": []}`), 0644); err != nil {
		t.Fatalf("failed to write registry.json: %v", err)
	}

	r, err := LoadRegistry(dir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(r.Skills) != 0 {
		t.Fatalf("expected 0 skills, got %d", len(r.Skills))
	}
}

// TestInstalledStatus_AllInstalled verifies matching versions produce "installed".
func TestInstalledStatus_AllInstalled(t *testing.T) {
	reg := &Registry{
		Skills: []RegistryEntry{
			{Name: "skill-a", Version: "v1.0.0"},
		},
	}
	loaded := []Skill{
		{Name: "skill-a", Version: "v1.0.0"},
	}

	status := reg.InstalledStatus(loaded)
	if status["skill-a"] != "installed" {
		t.Fatalf("expected 'installed', got %q", status["skill-a"])
	}
}

// TestInstalledStatus_NotInstalled verifies missing loaded skill produces "not-installed".
func TestInstalledStatus_NotInstalled(t *testing.T) {
	reg := &Registry{
		Skills: []RegistryEntry{
			{Name: "skill-a", Version: "v1.0.0"},
		},
	}

	status := reg.InstalledStatus(nil)
	if status["skill-a"] != "not-installed" {
		t.Fatalf("expected 'not-installed', got %q", status["skill-a"])
	}
}

// TestInstalledStatus_UpdateAvailable verifies differing versions produce "update-available".
func TestInstalledStatus_UpdateAvailable(t *testing.T) {
	reg := &Registry{
		Skills: []RegistryEntry{
			{Name: "skill-a", Version: "v2.0.0"},
		},
	}
	loaded := []Skill{
		{Name: "skill-a", Version: "v1.0.0"},
	}

	status := reg.InstalledStatus(loaded)
	if status["skill-a"] != "update-available" {
		t.Fatalf("expected 'update-available', got %q", status["skill-a"])
	}
}

// TestInstalledStatus_NoVersion verifies empty versions produce "installed" (no comparison possible).
func TestInstalledStatus_NoVersion(t *testing.T) {
	reg := &Registry{
		Skills: []RegistryEntry{
			{Name: "skill-a", Version: ""},
		},
	}
	loaded := []Skill{
		{Name: "skill-a", Version: ""},
	}

	status := reg.InstalledStatus(loaded)
	if status["skill-a"] != "installed" {
		t.Fatalf("expected 'installed', got %q", status["skill-a"])
	}
}

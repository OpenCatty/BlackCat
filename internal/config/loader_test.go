package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoad_ExplicitPath tests Load with an explicit valid file path.
func TestLoad_ExplicitPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yaml := `
server:
  addr: ":9999"
  port: 9999
opencode:
  addr: "http://custom:5000"
  timeout: "5m"
llm:
  provider: "openai"
  model: "gpt-4"
`

	if err := os.WriteFile(configPath, []byte(yaml), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify explicit path loaded correctly
	if cfg.Server.Port != 9999 {
		t.Errorf("Server.Port = %d, want 9999", cfg.Server.Port)
	}
	if cfg.OpenCode.Addr != "http://custom:5000" {
		t.Errorf("OpenCode.Addr = %q, want %q", cfg.OpenCode.Addr, "http://custom:5000")
	}
	if cfg.OpenCode.Timeout.Duration != 5*time.Minute {
		t.Errorf("OpenCode.Timeout = %v, want %v", cfg.OpenCode.Timeout.Duration, 5*time.Minute)
	}
	if cfg.LLM.Provider != "openai" {
		t.Errorf("LLM.Provider = %q, want %q", cfg.LLM.Provider, "openai")
	}
}

// TestLoad_EmptyPath_NoFile tests Load with empty path when no config file exists.
// Should return defaults without error.
func TestLoad_EmptyPath_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temporary directory where no blackcat.yaml exists
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(origCwd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Load with empty path - should use defaults since no file exists
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify defaults are applied
	if cfg.Server.Addr != ":8080" {
		t.Errorf("Server.Addr = %q, want %q", cfg.Server.Addr, ":8080")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.OpenCode.Addr != "http://127.0.0.1:4096" {
		t.Errorf("OpenCode.Addr = %q, want %q", cfg.OpenCode.Addr, "http://127.0.0.1:4096")
	}
	if cfg.OpenCode.Timeout.Duration != 30*time.Minute {
		t.Errorf("OpenCode.Timeout = %v, want %v", cfg.OpenCode.Timeout.Duration, 30*time.Minute)
	}
	if cfg.LLM.Temperature != 0.7 {
		t.Errorf("LLM.Temperature = %f, want 0.7", cfg.LLM.Temperature)
	}
	if cfg.Memory.FilePath != "MEMORY.md" {
		t.Errorf("Memory.FilePath = %q, want %q", cfg.Memory.FilePath, "MEMORY.md")
	}
}

// TestLoad_AutoDiscovery tests auto-discovery when Load("") is called.
// Should find and load blackcat.yaml in the current working directory.
func TestLoad_AutoDiscovery(t *testing.T) {
	tmpDir := t.TempDir()

	// Write config file to temp directory
	configPath := filepath.Join(tmpDir, "blackcat.yaml")
	yaml := `
server:
  port: 7777
opencode:
  addr: "http://discovery:9000"
  timeout: "15m"
memory:
  filePath: "custom_memory.md"
  consolidationThreshold: 25
`

	if err := os.WriteFile(configPath, []byte(yaml), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Change to temp directory
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(origCwd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Load with empty path - should auto-discover blackcat.yaml
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify auto-discovered config was loaded
	if cfg.Server.Port != 7777 {
		t.Errorf("Server.Port = %d, want 7777", cfg.Server.Port)
	}
	if cfg.OpenCode.Addr != "http://discovery:9000" {
		t.Errorf("OpenCode.Addr = %q, want %q", cfg.OpenCode.Addr, "http://discovery:9000")
	}
	if cfg.OpenCode.Timeout.Duration != 15*time.Minute {
		t.Errorf("OpenCode.Timeout = %v, want %v", cfg.OpenCode.Timeout.Duration, 15*time.Minute)
	}
	if cfg.Memory.FilePath != "custom_memory.md" {
		t.Errorf("Memory.FilePath = %q, want %q", cfg.Memory.FilePath, "custom_memory.md")
	}
	if cfg.Memory.ConsolidationThreshold != 25 {
		t.Errorf("Memory.ConsolidationThreshold = %d, want 25", cfg.Memory.ConsolidationThreshold)
	}
}

// TestLoad_AutoDiscovery_WithDefaults tests that defaults are applied
// alongside auto-discovered config values.
func TestLoad_AutoDiscovery_WithDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	// Write minimal config file (only specify one field)
	configPath := filepath.Join(tmpDir, "blackcat.yaml")
	yaml := `
server:
  port: 8888
`

	if err := os.WriteFile(configPath, []byte(yaml), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Change to temp directory
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(origCwd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Load with auto-discovery
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify loaded config and defaults are combined
	if cfg.Server.Port != 8888 {
		t.Errorf("Server.Port = %d, want 8888", cfg.Server.Port)
	}
	// Default should still be applied for unspecified field
	if cfg.OpenCode.Addr != "http://127.0.0.1:4096" {
		t.Errorf("OpenCode.Addr = %q, want %q (should be default)", cfg.OpenCode.Addr, "http://127.0.0.1:4096")
	}
	// LLM defaults should apply
	if cfg.LLM.Temperature != 0.7 {
		t.Errorf("LLM.Temperature = %f, want 0.7 (default)", cfg.LLM.Temperature)
	}
}

// TestLoad_ConfigWithDuration tests that Duration fields are correctly unmarshalled.
func TestLoad_ConfigWithDuration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yaml := `
opencode:
  addr: "http://localhost:4096"
  timeout: "45m"
`

	if err := os.WriteFile(configPath, []byte(yaml), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.OpenCode.Timeout.Duration != 45*time.Minute {
		t.Errorf("OpenCode.Timeout = %v, want %v", cfg.OpenCode.Timeout.Duration, 45*time.Minute)
	}
}

// TestLoad_DurationAsInteger tests Duration field unmarshalling from integer (seconds).
func TestLoad_DurationAsInteger(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yaml := `
opencode:
  addr: "http://localhost:4096"
  timeout: 3600
`

	if err := os.WriteFile(configPath, []byte(yaml), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.OpenCode.Timeout.Duration != 3600*time.Second {
		t.Errorf("OpenCode.Timeout = %v, want %v", cfg.OpenCode.Timeout.Duration, 3600*time.Second)
	}
}

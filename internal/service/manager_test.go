package service

import (
	"testing"
)

func TestServiceConfigValidate_EmptyName(t *testing.T) {
	cfg := ServiceConfig{Name: "", BinaryPath: "/some/path"}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should fail with empty name")
	}
}

func TestServiceConfigValidate_EmptyBinaryPath(t *testing.T) {
	cfg := ServiceConfig{Name: "blackcat", BinaryPath: ""}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should fail with empty binary path")
	}
}

func TestServiceConfigValidate_NonExistentBinary(t *testing.T) {
	cfg := ServiceConfig{Name: "blackcat", BinaryPath: "/nonexistent/binary/path"}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should fail with non-existent binary")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Name != "blackcat" {
		t.Errorf("DefaultConfig().Name = %q, want %q", cfg.Name, "blackcat")
	}
	if cfg.DisplayName == "" {
		t.Error("DefaultConfig().DisplayName is empty")
	}
	if cfg.Description == "" {
		t.Error("DefaultConfig().Description is empty")
	}
	if cfg.ConfigPath == "" {
		t.Error("DefaultConfig().ConfigPath is empty")
	}
	if cfg.WorkDir == "" {
		t.Error("DefaultConfig().WorkDir is empty")
	}
}

func TestNew(t *testing.T) {
	mgr := New()
	if mgr == nil {
		t.Fatal("New() returned nil")
	}
}

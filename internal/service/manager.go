// Package service provides cross-platform daemon management for BlackCat.
// It abstracts systemd (Linux), launchd (macOS), and schtasks (Windows)
// behind a common Manager interface, always using user-level services (no root).
package service

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ServiceConfig holds the configuration for installing the daemon service.
type ServiceConfig struct {
	Name        string // service name, e.g. "blackcat"
	DisplayName string // human-readable name, e.g. "BlackCat Daemon"
	Description string // service description
	BinaryPath  string // absolute path to the blackcat binary
	ConfigPath  string // path to config file (--config flag value)
	WorkDir     string // working directory for the service
	VaultPass   string // vault passphrase (set as environment variable)
}

// ServiceStatus represents the current state of the daemon service.
type ServiceStatus struct {
	Installed bool
	Running   bool
	PID       int
	Uptime    time.Duration
}

// Manager provides cross-platform daemon lifecycle management.
// All operations use user-level services (no root/sudo required).
type Manager interface {
	// Install registers the daemon as a user-level service.
	Install(cfg ServiceConfig) error
	// Uninstall removes the service registration.
	Uninstall() error
	// Start starts the daemon service.
	Start() error
	// Stop stops the daemon service.
	Stop() error
	// Restart stops and starts the daemon service.
	Restart() error
	// Status returns the current service state.
	Status() (ServiceStatus, error)
	// IsInstalled returns true if the service is registered.
	IsInstalled() bool
}

// Validate checks that required ServiceConfig fields are set.
func (c *ServiceConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("service name is required")
	}
	if c.BinaryPath == "" {
		return fmt.Errorf("binary path is required")
	}
	if _, err := os.Stat(c.BinaryPath); err != nil {
		return fmt.Errorf("binary not found at %s: %w", c.BinaryPath, err)
	}
	return nil
}

// DefaultConfig returns a ServiceConfig with sensible defaults for BlackCat.
func DefaultConfig() ServiceConfig {
	home, _ := os.UserHomeDir()
	binPath, _ := os.Executable()
	return ServiceConfig{
		Name:        "blackcat",
		DisplayName: "BlackCat Daemon",
		Description: "BlackCat AI Agent Daemon",
		BinaryPath:  binPath,
		ConfigPath:  filepath.Join(home, ".blackcat", "config.yaml"),
		WorkDir:     filepath.Join(home, ".blackcat"),
	}
}

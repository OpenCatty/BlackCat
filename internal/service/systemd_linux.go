//go:build linux

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const serviceName = "blackcat"

// systemdManager implements Manager using systemd --user (user-level, no root).
type systemdManager struct{}

// New returns a systemd-based Manager for Linux.
func New() Manager {
	return &systemdManager{}
}

func (m *systemdManager) unitPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", serviceName+".service")
}

func (m *systemdManager) Install(cfg ServiceConfig) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	unitDir := filepath.Dir(m.unitPath())
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		return fmt.Errorf("failed to create systemd user dir: %w", err)
	}

	execStart := cfg.BinaryPath + " daemon"
	if cfg.ConfigPath != "" {
		execStart += " --config " + cfg.ConfigPath
	}

	unit := fmt.Sprintf(`[Unit]
Description=%s
After=network-online.target

[Service]
Type=simple
ExecStart=%s
WorkingDirectory=%s
Restart=on-failure
RestartSec=5
`, cfg.Description, execStart, cfg.WorkDir)

	if cfg.VaultPass != "" {
		unit += fmt.Sprintf("Environment=BLACKCAT_VAULT_PASSPHRASE=%s\n", cfg.VaultPass)
	}

	unit += `
[Install]
WantedBy=default.target
`

	if err := os.WriteFile(m.unitPath(), []byte(unit), 0o644); err != nil {
		return fmt.Errorf("failed to write unit file: %w", err)
	}

	if err := systemctl("daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload failed: %w", err)
	}
	if err := systemctl("enable", serviceName); err != nil {
		return fmt.Errorf("enable failed: %w", err)
	}

	return nil
}

func (m *systemdManager) Uninstall() error {
	if !m.IsInstalled() {
		return fmt.Errorf("service is not installed")
	}

	st, _ := m.Status()
	if st.Running {
		_ = m.Stop()
	}

	_ = systemctl("disable", serviceName)
	if err := os.Remove(m.unitPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove unit file: %w", err)
	}
	_ = systemctl("daemon-reload")
	return nil
}

func (m *systemdManager) Start() error {
	return systemctl("start", serviceName)
}

func (m *systemdManager) Stop() error {
	return systemctl("stop", serviceName)
}

func (m *systemdManager) Restart() error {
	return systemctl("restart", serviceName)
}

func (m *systemdManager) Status() (ServiceStatus, error) {
	st := ServiceStatus{}
	if !m.IsInstalled() {
		return st, nil
	}
	st.Installed = true

	out, err := exec.Command("systemctl", "--user", "show", serviceName,
		"--property=ActiveState,MainPID,ExecMainStartTimestamp").Output()
	if err != nil {
		return st, fmt.Errorf("systemctl show failed: %w", err)
	}

	props := parseProperties(string(out))
	if props["ActiveState"] == "active" {
		st.Running = true
	}
	if pidStr, ok := props["MainPID"]; ok {
		st.PID, _ = strconv.Atoi(pidStr)
	}
	if tsStr, ok := props["ExecMainStartTimestamp"]; ok && tsStr != "" {
		if t, err := time.Parse("Mon 2006-01-02 15:04:05 MST", tsStr); err == nil {
			st.Uptime = time.Since(t)
		}
	}

	return st, nil
}

func (m *systemdManager) IsInstalled() bool {
	_, err := os.Stat(m.unitPath())
	return err == nil
}

func systemctl(args ...string) error {
	cmdArgs := append([]string{"--user"}, args...)
	cmd := exec.Command("systemctl", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func parseProperties(output string) map[string]string {
	props := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, "="); idx > 0 {
			props[line[:idx]] = line[idx+1:]
		}
	}
	return props
}

//go:build windows

package service

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	taskName    = "BlackCat Daemon"
	serviceName = "blackcat"
)

// schtasksManager implements Manager using Windows Task Scheduler (user-level, no admin).
type schtasksManager struct{}

// New returns a schtasks-based Manager for Windows.
func New() Manager {
	return &schtasksManager{}
}

func (m *schtasksManager) Install(cfg ServiceConfig) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	tr := cfg.BinaryPath + " daemon"
	if cfg.ConfigPath != "" {
		tr += " --config " + cfg.ConfigPath
	}

	args := []string{
		"/Create",
		"/SC", "ONLOGON",
		"/TN", taskName,
		"/TR", tr,
		"/F", // force overwrite if exists
	}

	cmd := exec.Command("schtasks", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("schtasks create failed: %w", err)
	}

	return nil
}

func (m *schtasksManager) Uninstall() error {
	if !m.IsInstalled() {
		return fmt.Errorf("service is not installed")
	}

	st, _ := m.Status()
	if st.Running {
		_ = m.Stop()
	}

	cmd := exec.Command("schtasks", "/Delete", "/TN", taskName, "/F")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("schtasks delete failed: %w", err)
	}
	return nil
}

func (m *schtasksManager) Start() error {
	cmd := exec.Command("schtasks", "/Run", "/TN", taskName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *schtasksManager) Stop() error {
	cmd := exec.Command("taskkill", "/IM", "blackcat.exe", "/F")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *schtasksManager) Restart() error {
	_ = m.Stop()
	return m.Start()
}

func (m *schtasksManager) Status() (ServiceStatus, error) {
	st := ServiceStatus{}
	if !m.IsInstalled() {
		return st, nil
	}
	st.Installed = true

	// Check if blackcat.exe is running via tasklist
	out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq blackcat.exe", "/FO", "CSV", "/NH").Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			if strings.Contains(line, "blackcat.exe") {
				// CSV format: "blackcat.exe","1234","Console","1","12,345 K"
				fields := strings.Split(line, ",")
				if len(fields) >= 2 {
					pidStr := strings.Trim(fields[1], "\"")
					if pid, err := strconv.Atoi(pidStr); err == nil {
						st.Running = true
						st.PID = pid
					}
				}
				break
			}
		}
	}

	return st, nil
}

func (m *schtasksManager) IsInstalled() bool {
	cmd := exec.Command("schtasks", "/Query", "/TN", taskName)
	return cmd.Run() == nil
}

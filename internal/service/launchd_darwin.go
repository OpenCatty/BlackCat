//go:build darwin

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	serviceLabel = "ai.blackcat.daemon"
	serviceName  = "blackcat"
)

// launchdManager implements Manager using macOS LaunchAgent (user-level, no root).
type launchdManager struct{}

// New returns a launchd-based Manager for macOS.
func New() Manager {
	return &launchdManager{}
}

func (m *launchdManager) plistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", serviceLabel+".plist")
}

func (m *launchdManager) Install(cfg ServiceConfig) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	plistDir := filepath.Dir(m.plistPath())
	if err := os.MkdirAll(plistDir, 0o755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents dir: %w", err)
	}

	args := []string{"daemon"}
	if cfg.ConfigPath != "" {
		args = append(args, "--config", cfg.ConfigPath)
	}

	programArgs := fmt.Sprintf("\t\t<string>%s</string>\n", cfg.BinaryPath)
	for _, arg := range args {
		programArgs += fmt.Sprintf("\t\t<string>%s</string>\n", arg)
	}

	envSection := ""
	if cfg.VaultPass != "" {
		envSection = fmt.Sprintf(`	<key>EnvironmentVariables</key>
	<dict>
		<key>BLACKCAT_VAULT_PASSPHRASE</key>
		<string>%s</string>
	</dict>
`, cfg.VaultPass)
	}

	logDir := cfg.WorkDir
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
%s	</array>
	<key>WorkingDirectory</key>
	<string>%s</string>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
%s</dict>
</plist>
`, serviceLabel, programArgs, cfg.WorkDir,
		filepath.Join(logDir, "blackcat.log"),
		filepath.Join(logDir, "blackcat.err.log"),
		envSection)

	if err := os.WriteFile(m.plistPath(), []byte(plist), 0o644); err != nil {
		return fmt.Errorf("failed to write plist: %w", err)
	}

	if err := launchctl("load", m.plistPath()); err != nil {
		return fmt.Errorf("launchctl load failed: %w", err)
	}

	return nil
}

func (m *launchdManager) Uninstall() error {
	if !m.IsInstalled() {
		return fmt.Errorf("service is not installed")
	}

	_ = launchctl("unload", m.plistPath())

	if err := os.Remove(m.plistPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist: %w", err)
	}
	return nil
}

func (m *launchdManager) Start() error {
	return launchctl("start", serviceLabel)
}

func (m *launchdManager) Stop() error {
	return launchctl("stop", serviceLabel)
}

func (m *launchdManager) Restart() error {
	_ = m.Stop()
	return m.Start()
}

func (m *launchdManager) Status() (ServiceStatus, error) {
	st := ServiceStatus{}
	if !m.IsInstalled() {
		return st, nil
	}
	st.Installed = true

	out, err := exec.Command("launchctl", "list").Output()
	if err != nil {
		return st, fmt.Errorf("launchctl list failed: %w", err)
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, serviceLabel) {
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				if pid, err := strconv.Atoi(fields[0]); err == nil && pid > 0 {
					st.Running = true
					st.PID = pid
				}
			}
			break
		}
	}

	return st, nil
}

func (m *launchdManager) IsInstalled() bool {
	_, err := os.Stat(m.plistPath())
	return err == nil
}

func launchctl(args ...string) error {
	cmd := exec.Command("launchctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

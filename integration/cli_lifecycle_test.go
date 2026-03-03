//go:build integration

package integration

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var cliBinaryPath string

func TestMain(m *testing.M) {
	repoRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve working directory: %v\n", err)
		os.Exit(1)
	}
	repoRoot = filepath.Dir(repoRoot)

	buildDir, err := os.MkdirTemp("", "blackcat-cli-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create temp build dir: %v\n", err)
		os.Exit(1)
	}

	binaryName := "blackcat"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	cliBinaryPath = filepath.Join(buildDir, binaryName)

	buildCmd := exec.Command("go", "build", "-o", cliBinaryPath, ".")
	buildCmd.Dir = repoRoot
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build blackcat binary: %v\n%s", err, string(buildOutput))
		_ = os.RemoveAll(buildDir)
		os.Exit(1)
	}

	code := m.Run()

	_ = os.RemoveAll(buildDir)
	os.Exit(code)
}

func TestCLILifecycle(t *testing.T) {
	if os.Getenv("BLACKCAT_INTEGRATION") == "" {
		t.Skip("set BLACKCAT_INTEGRATION=1 to run")
	}
	if runtime.GOOS == "windows" {
		t.Skip("integration tests not supported on Windows")
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		t.Skip("systemctl not found")
	}

	configDir := t.TempDir()
	env := append(
		os.Environ(),
		"BLACKCAT_CONFIG_DIR="+configDir,
		"HOME="+configDir,
		"XDG_CONFIG_HOME="+filepath.Join(configDir, ".config"),
		"BLACKCAT_LLM_PROVIDER=openai",
		"BLACKCAT_LLM_APIKEY=test-key",
		"BLACKCAT_VAULT_PASSPHRASE=testpass",
		"BLACKCAT_CHANNELS_TELEGRAM_ENABLED=true",
		"BLACKCAT_CHANNELS_TELEGRAM_TOKEN=test-token",
	)

	t.Cleanup(func() {
		exitCode, output := runCLI(env, "uninstall", "--yes")
		if exitCode != 0 {
			t.Logf("cleanup uninstall failed (exit=%d):\n%s", exitCode, output)
		}
	})

	t.Run("step-1-doctor", func(t *testing.T) {
		exitCode, output := runCLI(env, "doctor")
		if exitCode != 0 && !strings.Contains(output, "checks failed") {
			t.Fatalf("doctor failed unexpectedly (exit=%d):\n%s", exitCode, output)
		}
		if strings.TrimSpace(output) == "" {
			t.Fatalf("doctor produced no output")
		}
	})

	t.Run("step-2-onboard", func(t *testing.T) {
		exitCode, output := runCLI(env, "onboard", "--non-interactive")
		if exitCode != 0 {
			t.Fatalf("onboard failed (exit=%d):\n%s", exitCode, output)
		}
	})

	t.Run("step-3-status", func(t *testing.T) {
		statusOutput := waitForCommandOutput(t, 20*time.Second, 2*time.Second, env, "status")
		if !strings.Contains(strings.ToLower(statusOutput), "running") {
			t.Fatalf("status output does not indicate running:\n%s", statusOutput)
		}
	})

	t.Run("step-4-channels-list", func(t *testing.T) {
		exitCode, output := runCLI(env, "channels", "list")
		if exitCode != 0 {
			t.Fatalf("channels list failed (exit=%d):\n%s", exitCode, output)
		}
		if !strings.Contains(output, "CHANNEL") {
			t.Fatalf("channels list output missing header:\n%s", output)
		}
		if !strings.Contains(output, "telegram") || !strings.Contains(output, "discord") || !strings.Contains(output, "whatsapp") {
			t.Fatalf("channels list output missing expected channels:\n%s", output)
		}
	})

	t.Run("step-5-restart", func(t *testing.T) {
		exitCode, output := runCLI(env, "restart")
		if exitCode != 0 {
			t.Fatalf("restart failed (exit=%d):\n%s", exitCode, output)
		}
	})

	t.Run("step-6-stop", func(t *testing.T) {
		exitCode, output := runCLI(env, "stop")
		if exitCode != 0 {
			t.Fatalf("stop failed (exit=%d):\n%s", exitCode, output)
		}
		if !strings.Contains(strings.ToLower(output), "stopped") {
			t.Fatalf("stop output does not indicate stopped:\n%s", output)
		}
	})

	t.Run("step-7-uninstall", func(t *testing.T) {
		exitCode, output := runCLI(env, "uninstall", "--yes")
		if exitCode != 0 {
			t.Fatalf("uninstall failed (exit=%d):\n%s", exitCode, output)
		}
	})
}

func waitForCommandOutput(t *testing.T, timeout time.Duration, interval time.Duration, env []string, args ...string) string {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var lastOutput string
	for {
		exitCode, output := runCLI(env, args...)
		lastOutput = output
		if exitCode == 0 {
			return output
		}
		if time.Now().After(deadline) {
			t.Fatalf("command %q failed until timeout (last exit=%d):\n%s", strings.Join(args, " "), exitCode, lastOutput)
		}
		time.Sleep(interval)
	}
}

func runCLI(env []string, args ...string) (int, string) {
	cmd := exec.Command(cliBinaryPath, args...)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err == nil {
		return 0, string(output)
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), string(output)
	}

	return -1, fmt.Sprintf("failed to run %s: %v\n%s", strings.Join(args, " "), err, string(output))
}

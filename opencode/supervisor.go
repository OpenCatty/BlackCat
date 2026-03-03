// Package opencode — supervisor.go manages the opencode serve process.
package opencode

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// SupervisorConfig configures the OpenCode server process.
type SupervisorConfig struct {
	// Binary is the path or name of the opencode binary. Defaults to "opencode".
	Binary string
	// Port is the HTTP port to bind. Defaults to 4096.
	Port int
	// Dir is the working directory for opencode. Defaults to current directory.
	Dir string
	// Password is the optional OPENCODE_SERVER_PASSWORD value.
	Password string
	// StartTimeout is how long to wait for the server to become healthy.
	StartTimeout time.Duration
	// ExtraArgs are additional CLI args passed to "opencode serve".
	ExtraArgs []string
}

// Supervisor manages the lifecycle of an opencode serve child process.
type Supervisor struct {
	cfg SupervisorConfig
	cmd *exec.Cmd
}

// NewSupervisor creates a Supervisor with the given configuration.
func NewSupervisor(cfg SupervisorConfig) *Supervisor {
	if cfg.Binary == "" {
		cfg.Binary = "opencode"
	}
	if cfg.Port == 0 {
		cfg.Port = 4096
	}
	if cfg.StartTimeout == 0 {
		cfg.StartTimeout = 30 * time.Second
	}
	return &Supervisor{cfg: cfg}
}

// Start launches "opencode serve" and blocks until the server is healthy.
func (s *Supervisor) Start(ctx context.Context) error {
	args := []string{"serve", "--port", fmt.Sprintf("%d", s.cfg.Port)}
	if s.cfg.Dir != "" {
		args = append(args, "--dir", s.cfg.Dir)
	}
	args = append(args, s.cfg.ExtraArgs...)

	s.cmd = exec.CommandContext(ctx, s.cfg.Binary, args...)
	// Inherit parent environment (required for AI provider API keys, PATH, etc.).
	s.cmd.Env = os.Environ()
	if s.cfg.Password != "" {
		s.cmd.Env = append(s.cmd.Env, "OPENCODE_SERVER_PASSWORD="+s.cfg.Password)
	}

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("starting opencode serve: %w", err)
	}

	// Poll the health endpoint until the server responds.
	deadline := time.Now().Add(s.cfg.StartTimeout)
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/global/health", s.cfg.Port)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		resp, err := http.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("opencode server did not become healthy within %s", s.cfg.StartTimeout)
}

// Stop kills the opencode child process.
func (s *Supervisor) Stop() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	_ = s.cmd.Process.Kill()
	_ = s.cmd.Wait()
	return nil
}

// BaseURL returns the HTTP base URL of the running opencode server.
func (s *Supervisor) BaseURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", s.cfg.Port)
}

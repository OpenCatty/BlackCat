package mcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"

	"github.com/startower-observability/blackcat/internal/skills"
)

// EmbeddedServer represents a lazily-started MCP server process.
type EmbeddedServer struct {
	Name    string
	Config  skills.MCPConfig
	Port    int
	started bool
	mu      sync.Mutex
	cmd     *exec.Cmd
	cancel  context.CancelFunc
}

// EmbeddedServerManager tracks all embedded MCP servers from skill frontmatter.
type EmbeddedServerManager struct {
	mu      sync.RWMutex
	servers map[string]*EmbeddedServer
}

// NewEmbeddedServerManager creates an EmbeddedServerManager with no servers registered.
func NewEmbeddedServerManager() *EmbeddedServerManager {
	return &EmbeddedServerManager{
		servers: make(map[string]*EmbeddedServer),
	}
}

// Register records an MCP server config but does NOT start the process (lazy start).
func (m *EmbeddedServerManager) Register(name string, cfg skills.MCPConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.servers[name] = &EmbeddedServer{
		Name:   name,
		Config: cfg,
	}
}

// Start lazily starts the embedded MCP server process for the given name.
// It is idempotent — calling Start() on an already-started server returns the existing server.
func (m *EmbeddedServerManager) Start(ctx context.Context, name string) (*EmbeddedServer, error) {
	m.mu.RLock()
	srv, ok := m.servers[name]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("embedded MCP: server %q not registered", name)
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()

	if srv.started {
		return srv, nil
	}

	if srv.Config.Command == "" {
		return nil, fmt.Errorf("embedded MCP: command required for server %q", name)
	}

	// Find a free port via OS assignment.
	port, err := findFreePort()
	if err != nil {
		return nil, fmt.Errorf("embedded MCP: find free port for %q: %w", name, err)
	}

	procCtx, cancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(procCtx, srv.Config.Command, srv.Config.Args...)
	cmd.Env = buildEnv(srv.Config.Env, port)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("embedded MCP: start server %q: %w", name, err)
	}

	srv.Port = port
	srv.cmd = cmd
	srv.cancel = cancel
	srv.started = true

	// Reap the process in the background so we don't leak zombies.
	go func() {
		_ = cmd.Wait()
	}()

	return srv, nil
}

// Get returns the EmbeddedServer by name and whether it exists.
func (m *EmbeddedServerManager) Get(name string) (*EmbeddedServer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	srv, ok := m.servers[name]
	return srv, ok
}

// IsStarted reports whether this server's process has been started.
func (s *EmbeddedServer) IsStarted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

// ListRunning returns the names of all currently running servers.
func (m *EmbeddedServerManager) ListRunning() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var running []string
	for name, srv := range m.servers {
		srv.mu.Lock()
		if srv.started {
			running = append(running, name)
		}
		srv.mu.Unlock()
	}
	return running
}

// StopAll stops all running embedded MCP server processes.
func (m *EmbeddedServerManager) StopAll(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, srv := range m.servers {
		srv.mu.Lock()
		if srv.started && srv.cancel != nil {
			srv.cancel()
			srv.started = false
			srv.cancel = nil
			srv.cmd = nil
		}
		srv.mu.Unlock()

		_ = name // suppress unused
	}

	return errors.Join(errs...)
}

// findFreePort asks the OS for an available TCP port.
func findFreePort() (int, error) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port, nil
}

// buildEnv constructs the environment slice for the child process,
// inheriting the current environment and adding config overrides plus MCP_PORT.
func buildEnv(env map[string]string, port int) []string {
	base := os.Environ()
	for k, v := range env {
		base = append(base, fmt.Sprintf("%s=%s", k, v))
	}
	base = append(base, fmt.Sprintf("MCP_PORT=%d", port))
	return base
}

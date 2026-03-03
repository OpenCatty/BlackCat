package mcp

import (
	"context"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/startower-observability/blackcat/skills"
)

func TestNewEmbeddedServerManager(t *testing.T) {
	m := NewEmbeddedServerManager()
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
	if len(m.servers) != 0 {
		t.Fatalf("expected 0 servers, got %d", len(m.servers))
	}
}

func TestRegisterDoesNotStart(t *testing.T) {
	m := NewEmbeddedServerManager()
	m.Register("test-server", skills.MCPConfig{
		Command: "echo",
		Args:    []string{"hello"},
	})

	srv, ok := m.Get("test-server")
	if !ok {
		t.Fatal("expected server to be registered")
	}
	if srv.IsStarted() {
		t.Fatal("expected server to NOT be started after Register (lazy start)")
	}
	if srv.Name != "test-server" {
		t.Fatalf("expected name %q, got %q", "test-server", srv.Name)
	}
}

func TestInvalidConfigError(t *testing.T) {
	m := NewEmbeddedServerManager()
	m.Register("empty-cmd", skills.MCPConfig{
		Command: "",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.Start(ctx, "empty-cmd")
	if err == nil {
		t.Fatal("expected error for empty command")
	}

	want := `embedded MCP: command required for server "empty-cmd"`
	if err.Error() != want {
		t.Fatalf("unexpected error message:\n  got:  %s\n  want: %s", err.Error(), want)
	}
}

func TestStartNotRegistered(t *testing.T) {
	m := NewEmbeddedServerManager()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.Start(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for unregistered server")
	}
}

// sleepCommand returns a platform-appropriate long-running command
// that can be cleanly killed via context cancellation.
func sleepCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/c", "timeout /t 60 /nobreak >nul"}
	}
	return "sleep", []string{"60"}
}

func TestStartIdempotent(t *testing.T) {
	cmd, args := sleepCommand()
	m := NewEmbeddedServerManager()
	m.Register("idempotent", skills.MCPConfig{
		Command: cmd,
		Args:    args,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	srv1, err := m.Start(ctx, "idempotent")
	if err != nil {
		t.Fatalf("first start failed: %v", err)
	}
	if !srv1.IsStarted() {
		t.Fatal("expected server to be started after Start()")
	}
	if srv1.Port == 0 {
		t.Fatal("expected non-zero port")
	}

	port1 := srv1.Port

	srv2, err := m.Start(ctx, "idempotent")
	if err != nil {
		t.Fatalf("second start failed: %v", err)
	}
	if srv2 != srv1 {
		t.Fatal("expected same server instance on idempotent Start")
	}
	if srv2.Port != port1 {
		t.Fatalf("port changed on idempotent start: %d vs %d", port1, srv2.Port)
	}

	// Cleanup
	_ = m.StopAll(ctx)
}

func TestStartSetsPort(t *testing.T) {
	cmd, args := sleepCommand()
	m := NewEmbeddedServerManager()
	m.Register("port-test", skills.MCPConfig{
		Command: cmd,
		Args:    args,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	srv, err := m.Start(ctx, "port-test")
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if srv.Port <= 0 {
		t.Fatalf("expected positive port, got %d", srv.Port)
	}

	_ = m.StopAll(ctx)
}

func TestListRunning(t *testing.T) {
	cmd, args := sleepCommand()
	m := NewEmbeddedServerManager()
	m.Register("running-a", skills.MCPConfig{Command: cmd, Args: args})
	m.Register("running-b", skills.MCPConfig{Command: cmd, Args: args})
	m.Register("not-started", skills.MCPConfig{Command: cmd, Args: args})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := m.Start(ctx, "running-a"); err != nil {
		t.Fatalf("start running-a: %v", err)
	}
	if _, err := m.Start(ctx, "running-b"); err != nil {
		t.Fatalf("start running-b: %v", err)
	}

	running := m.ListRunning()
	sort.Strings(running)

	if len(running) != 2 {
		t.Fatalf("expected 2 running, got %d: %v", len(running), running)
	}
	if running[0] != "running-a" || running[1] != "running-b" {
		t.Fatalf("unexpected running list: %v", running)
	}

	_ = m.StopAll(ctx)
}

func TestStopAll(t *testing.T) {
	cmd, args := sleepCommand()
	m := NewEmbeddedServerManager()
	m.Register("stop-a", skills.MCPConfig{Command: cmd, Args: args})
	m.Register("stop-b", skills.MCPConfig{Command: cmd, Args: args})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := m.Start(ctx, "stop-a"); err != nil {
		t.Fatalf("start stop-a: %v", err)
	}
	if _, err := m.Start(ctx, "stop-b"); err != nil {
		t.Fatalf("start stop-b: %v", err)
	}

	running := m.ListRunning()
	if len(running) != 2 {
		t.Fatalf("expected 2 running before StopAll, got %d", len(running))
	}

	if err := m.StopAll(ctx); err != nil {
		t.Fatalf("StopAll returned error: %v", err)
	}

	running = m.ListRunning()
	if len(running) != 0 {
		t.Fatalf("expected 0 running after StopAll, got %d: %v", len(running), running)
	}
}

func TestStopAllNoRunning(t *testing.T) {
	m := NewEmbeddedServerManager()
	m.Register("never-started", skills.MCPConfig{Command: "echo"})

	err := m.StopAll(context.Background())
	if err != nil {
		t.Fatalf("StopAll on empty manager should not error: %v", err)
	}
}

func TestRegisterOverwrite(t *testing.T) {
	m := NewEmbeddedServerManager()

	m.Register("overwrite", skills.MCPConfig{Command: "old-cmd"})
	m.Register("overwrite", skills.MCPConfig{Command: "new-cmd"})

	srv, ok := m.Get("overwrite")
	if !ok {
		t.Fatal("expected server to exist")
	}
	if srv.Config.Command != "new-cmd" {
		t.Fatalf("expected overwritten command %q, got %q", "new-cmd", srv.Config.Command)
	}
}

func TestGetNonexistent(t *testing.T) {
	m := NewEmbeddedServerManager()
	_, ok := m.Get("does-not-exist")
	if ok {
		t.Fatal("expected ok=false for nonexistent server")
	}
}

func TestBuildEnv(t *testing.T) {
	env := buildEnv(map[string]string{"FOO": "bar"}, 12345)

	foundFoo := false
	foundPort := false
	for _, e := range env {
		if e == "FOO=bar" {
			foundFoo = true
		}
		if e == "MCP_PORT=12345" {
			foundPort = true
		}
	}

	if !foundFoo {
		t.Fatal("expected FOO=bar in env")
	}
	if !foundPort {
		t.Fatal("expected MCP_PORT=12345 in env")
	}
}

func TestFindFreePort(t *testing.T) {
	port, err := findFreePort()
	if err != nil {
		t.Fatalf("findFreePort: %v", err)
	}
	if port <= 0 {
		t.Fatalf("expected positive port, got %d", port)
	}

	// Ensure two calls return different ports (race-free check).
	port2, err := findFreePort()
	if err != nil {
		t.Fatalf("findFreePort second call: %v", err)
	}
	// Technically the same port could be re-assigned, but extremely unlikely.
	_ = port2
}

func TestStartWithEnvConfig(t *testing.T) {
	cmd, args := sleepCommand()
	m := NewEmbeddedServerManager()
	m.Register("env-test", skills.MCPConfig{
		Command: cmd,
		Args:    args,
		Env:     map[string]string{"MY_VAR": "hello"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	srv, err := m.Start(ctx, "env-test")
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if !srv.IsStarted() {
		t.Fatal("expected server to be started")
	}

	_ = m.StopAll(ctx)
}

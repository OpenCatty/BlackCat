package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/startower-observability/blackcat/internal/config"
	"github.com/startower-observability/blackcat/internal/tools"
	"github.com/startower-observability/blackcat/internal/types"
)

var _ types.Tool = (*ProxyTool)(nil)

type testTool struct {
	name        string
	description string
	parameters  json.RawMessage
	exec        func(ctx context.Context, args json.RawMessage) (string, error)
}

func (t *testTool) Name() string {
	return t.name
}

func (t *testTool) Description() string {
	return t.description
}

func (t *testTool) Parameters() json.RawMessage {
	return t.parameters
}

func (t *testTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if t.exec == nil {
		return "", nil
	}
	return t.exec(ctx, args)
}

func TestNewServer(t *testing.T) {
	registry := tools.NewRegistry()
	srv := NewServer(registry)
	if srv == nil {
		t.Fatal("expected server")
	}
	if srv.server == nil {
		t.Fatal("expected mcp server")
	}
}

func TestServerToolConversion(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&testTool{
		name:        "alpha",
		description: "alpha tool",
		parameters:  json.RawMessage(`{"type":"object","properties":{"message":{"type":"string"}}}`),
	})
	registry.Register(&testTool{
		name:        "beta",
		description: "beta tool",
		parameters:  json.RawMessage(`{"type":"object","properties":{"count":{"type":"number"}}}`),
	})

	srv := NewServer(registry)
	toolsMap := srv.server.ListTools()
	if len(toolsMap) != 2 {
		t.Fatalf("expected 2 mcp tools, got %d", len(toolsMap))
	}

	alpha, ok := toolsMap["alpha"]
	if !ok {
		t.Fatal("expected alpha tool")
	}
	if alpha.Tool.Description != "alpha tool" {
		t.Fatalf("expected alpha description, got %q", alpha.Tool.Description)
	}

	beta, ok := toolsMap["beta"]
	if !ok {
		t.Fatal("expected beta tool")
	}
	if beta.Tool.Description != "beta tool" {
		t.Fatalf("expected beta description, got %q", beta.Tool.Description)
	}
}

func TestServerToolExecution(t *testing.T) {
	registry := tools.NewRegistry()
	called := false
	var got json.RawMessage

	registry.Register(&testTool{
		name:        "echo",
		description: "echo tool",
		parameters:  json.RawMessage(`{"type":"object","properties":{"value":{"type":"string"}}}`),
		exec: func(_ context.Context, args json.RawMessage) (string, error) {
			called = true
			got = append(json.RawMessage(nil), args...)
			return "ok", nil
		},
	})

	srv := NewServer(registry)
	serverTool := srv.server.GetTool("echo")
	if serverTool == nil {
		t.Fatal("expected mcp tool handler")
	}

	result, err := serverTool.Handler(context.Background(), mcpgo.CallToolRequest{
		Params: mcpgo.CallToolParams{
			Name:      "echo",
			Arguments: map[string]any{"value": "hello"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected non-error result")
	}
	if !called {
		t.Fatal("expected registry execute to be called")
	}
	if string(got) != `{"value":"hello"}` {
		t.Fatalf("unexpected args passed to registry: %s", string(got))
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("expected client")
	}
	if len(client.Tools()) != 0 {
		t.Fatalf("expected no tools, got %d", len(client.Tools()))
	}
}

func TestProxyTool(t *testing.T) {
	client := NewClient()
	proxy := &ProxyTool{
		name:        "external_echo",
		description: "external echo",
		parameters:  json.RawMessage(`{"type":"object"}`),
		serverName:  "external",
		client:      client,
	}

	if proxy.Name() != "external_echo" {
		t.Fatalf("unexpected name: %q", proxy.Name())
	}
	if proxy.Description() != "external echo" {
		t.Fatalf("unexpected description: %q", proxy.Description())
	}
	if string(proxy.Parameters()) != `{"type":"object"}` {
		t.Fatalf("unexpected parameters: %s", string(proxy.Parameters()))
	}

	_, err := proxy.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected execute to fail without server connection")
	}
}

func TestClientConnectInvalidCommand(t *testing.T) {
	client := NewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := client.Connect(ctx, config.MCPServerConfig{
		Name:    "invalid",
		Command: "this-command-should-not-exist-interstellar-tests",
	})
	if err == nil {
		t.Fatal("expected connect error for invalid command")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected immediate command error, got timeout: %v", err)
	}
}

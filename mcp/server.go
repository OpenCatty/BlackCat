package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/startower-observability/blackcat/tools"
	"github.com/startower-observability/blackcat/types"
)

type Server struct {
	registry *tools.Registry
	server   *mcpserver.MCPServer
}

func NewServer(registry *tools.Registry) *Server {
	if registry == nil {
		registry = tools.NewRegistry()
	}

	s := &Server{
		registry: registry,
		server:   mcpserver.NewMCPServer("blackcat", "1.0.0", mcpserver.WithToolCapabilities(true)),
	}

	for _, def := range registry.List() {
		toolName := def.Name
		s.server.AddTool(convertTool(def), func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
			return handleToolCall(ctx, registry, toolName, request.GetRawArguments())
		})
	}

	return s
}

func (s *Server) ServeStdio(ctx context.Context) error {
	stdio := mcpserver.NewStdioServer(s.server)
	return stdio.Listen(ctx, os.Stdin, os.Stdout)
}

func (s *Server) ServeHTTP(ctx context.Context, addr string) error {
	httpServer := mcpserver.NewStreamableHTTPServer(s.server)
	errCh := make(chan error, 1)

	go func() {
		errCh <- httpServer.Start(addr)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		shutdownErr := httpServer.Shutdown(shutdownCtx)
		startErr := <-errCh
		if startErr != nil && !errors.Is(startErr, http.ErrServerClosed) {
			return startErr
		}
		if shutdownErr != nil {
			return shutdownErr
		}
		return nil
	}
}

func convertTool(t types.ToolDefinition) mcpgo.Tool {
	schema := t.Parameters
	if len(schema) == 0 {
		schema = json.RawMessage(`{"type":"object","properties":{}}`)
	}

	return mcpgo.NewToolWithRawSchema(t.Name, t.Description, schema)
}

func handleToolCall(ctx context.Context, registry *tools.Registry, toolName string, args any) (*mcpgo.CallToolResult, error) {
	rawArgs, err := toRawJSON(args)
	if err != nil {
		return mcpgo.NewToolResultErrorFromErr("invalid tool arguments", err), nil
	}

	result, err := registry.Execute(ctx, toolName, rawArgs)
	if err != nil {
		return mcpgo.NewToolResultErrorFromErr(fmt.Sprintf("tool %q execution failed", toolName), err), nil
	}

	return mcpgo.NewToolResultText(result), nil
}

func toRawJSON(v any) (json.RawMessage, error) {
	if v == nil {
		return json.RawMessage(`{}`), nil
	}

	if raw, ok := v.(json.RawMessage); ok {
		if len(raw) == 0 {
			return json.RawMessage(`{}`), nil
		}
		if !json.Valid(raw) {
			return nil, fmt.Errorf("invalid json payload")
		}
		return raw, nil
	}

	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return json.RawMessage(`{}`), nil
	}

	return json.RawMessage(b), nil
}

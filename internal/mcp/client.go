package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	mcpclient "github.com/mark3labs/mcp-go/client"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/startower-observability/blackcat/internal/config"
	"github.com/startower-observability/blackcat/internal/types"
)

type Client struct {
	mu      sync.RWMutex
	servers map[string]*connectedServer
	tools   []types.ToolDefinition
}

type connectedServer struct {
	client *mcpclient.Client
	tools  map[string]types.ToolDefinition
}

type ProxyTool struct {
	name        string
	description string
	parameters  json.RawMessage
	serverName  string
	client      *Client
}

func NewClient() *Client {
	return &Client{
		servers: make(map[string]*connectedServer),
		tools:   make([]types.ToolDefinition, 0),
	}
}

func (c *Client) Connect(ctx context.Context, cfg config.MCPServerConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("mcp server name is required")
	}
	if cfg.Command == "" {
		return fmt.Errorf("mcp server command is required")
	}

	env := envMapToSlice(cfg.Env)
	cli, err := mcpclient.NewStdioMCPClientWithOptions(cfg.Command, env, cfg.Args)
	if err != nil {
		return fmt.Errorf("start mcp server %q: %w", cfg.Name, err)
	}

	initReq := mcpgo.InitializeRequest{
		Params: mcpgo.InitializeParams{
			ProtocolVersion: mcpgo.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcpgo.Implementation{
				Name:    "blackcat-mcp-client",
				Version: "1.0.0",
			},
			Capabilities: mcpgo.ClientCapabilities{},
		},
	}

	if _, err := cli.Initialize(ctx, initReq); err != nil {
		_ = cli.Close()
		return fmt.Errorf("initialize mcp server %q: %w", cfg.Name, err)
	}

	listResult, err := cli.ListTools(ctx, mcpgo.ListToolsRequest{})
	if err != nil {
		_ = cli.Close()
		return fmt.Errorf("list tools from mcp server %q: %w", cfg.Name, err)
	}

	serverTools := make(map[string]types.ToolDefinition, len(listResult.Tools))
	discovered := make([]types.ToolDefinition, 0, len(listResult.Tools))
	for _, tool := range listResult.Tools {
		params, err := toolSchema(tool)
		if err != nil {
			_ = cli.Close()
			return fmt.Errorf("parse tool schema for %q on %q: %w", tool.Name, cfg.Name, err)
		}

		def := types.ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  params,
		}
		serverTools[tool.Name] = def
		discovered = append(discovered, def)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if existing, ok := c.servers[cfg.Name]; ok {
		_ = existing.client.Close()
	}

	c.servers[cfg.Name] = &connectedServer{
		client: cli,
		tools:  serverTools,
	}
	c.refreshToolsLocked()

	return nil
}

func (c *Client) ConnectAll(ctx context.Context, configs []config.MCPServerConfig) error {
	for _, cfg := range configs {
		if err := c.Connect(ctx, cfg); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) Tools() []types.ToolDefinition {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]types.ToolDefinition, len(c.tools))
	copy(out, c.tools)
	return out
}

func (c *Client) Execute(ctx context.Context, serverName, toolName string, args json.RawMessage) (string, error) {
	c.mu.RLock()
	serverConn, ok := c.servers[serverName]
	c.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("mcp server %q not connected", serverName)
	}

	arguments, err := decodeArguments(args)
	if err != nil {
		return "", fmt.Errorf("decode tool arguments: %w", err)
	}

	result, err := serverConn.client.CallTool(ctx, mcpgo.CallToolRequest{
		Params: mcpgo.CallToolParams{
			Name:      toolName,
			Arguments: arguments,
		},
	})
	if err != nil {
		return "", err
	}

	text := renderCallToolResult(result)
	if result.IsError {
		if text == "" {
			text = "tool returned an error"
		}
		return "", errors.New(text)
	}

	return text, nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	errList := make([]error, 0, len(c.servers))
	for _, serverConn := range c.servers {
		if err := serverConn.client.Close(); err != nil {
			errList = append(errList, err)
		}
	}

	c.servers = make(map[string]*connectedServer)
	c.tools = nil

	return errors.Join(errList...)
}

func (p *ProxyTool) Name() string {
	return p.name
}

func (p *ProxyTool) Description() string {
	return p.description
}

func (p *ProxyTool) Parameters() json.RawMessage {
	return p.parameters
}

func (p *ProxyTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return p.client.Execute(ctx, p.serverName, p.name, args)
}

func (c *Client) refreshToolsLocked() {
	all := make([]types.ToolDefinition, 0)

	serverNames := slices.Collect(maps.Keys(c.servers))
	slices.Sort(serverNames)

	for _, serverName := range serverNames {
		server := c.servers[serverName]
		toolNames := slices.Collect(maps.Keys(server.tools))
		slices.Sort(toolNames)
		for _, toolName := range toolNames {
			all = append(all, server.tools[toolName])
		}
	}

	c.tools = all
}

func envMapToSlice(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}

	keys := slices.Collect(maps.Keys(env))
	slices.Sort(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%s=%s", k, env[k]))
	}
	return out
}

func toolSchema(tool mcpgo.Tool) (json.RawMessage, error) {
	if len(tool.RawInputSchema) != 0 {
		return tool.RawInputSchema, nil
	}

	b, err := json.Marshal(tool.InputSchema)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func decodeArguments(args json.RawMessage) (any, error) {
	if len(args) == 0 {
		return map[string]any{}, nil
	}

	var out any
	if err := json.Unmarshal(args, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func renderCallToolResult(result *mcpgo.CallToolResult) string {
	if result == nil {
		return ""
	}

	parts := make([]string, 0, len(result.Content))
	for _, content := range result.Content {
		text := strings.TrimSpace(mcpgo.GetTextFromContent(content))
		if text != "" {
			parts = append(parts, text)
		}
	}

	if len(parts) > 0 {
		return strings.Join(parts, "\n")
	}

	if result.StructuredContent != nil {
		b, err := json.Marshal(result.StructuredContent)
		if err == nil {
			return string(b)
		}
	}

	return ""
}

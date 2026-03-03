package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/startower-observability/blackcat/security"
	"github.com/startower-observability/blackcat/tools"
	"github.com/startower-observability/blackcat/types"
)

type loopMockLLM struct {
	responses []types.LLMResponse
	callIndex int
}

func (m *loopMockLLM) Chat(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (*types.LLMResponse, error) {
	_ = ctx
	_ = messages
	_ = tools

	if len(m.responses) == 0 {
		return &types.LLMResponse{}, nil
	}

	if m.callIndex >= len(m.responses) {
		last := m.responses[len(m.responses)-1]
		return &last, nil
	}

	resp := m.responses[m.callIndex]
	m.callIndex++
	return &resp, nil
}

func (m *loopMockLLM) Stream(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (<-chan types.Chunk, error) {
	_ = ctx
	_ = messages
	_ = tools
	ch := make(chan types.Chunk)
	close(ch)
	return ch, nil
}

type loopMockTool struct {
	name      string
	result    string
	callCount int
}

func (m *loopMockTool) Name() string { return m.name }

func (m *loopMockTool) Description() string { return "mock tool" }

func (m *loopMockTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }

func (m *loopMockTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	_ = ctx
	_ = args
	m.callCount++
	return m.result, nil
}

func TestAgentLoop(t *testing.T) {
	ctx := context.Background()
	registry := tools.NewRegistry()
	tool := &loopMockTool{name: "mock_tool", result: "tool output"}
	registry.Register(tool)

	llm := &loopMockLLM{responses: []types.LLMResponse{
		{ToolCalls: []types.ToolCall{{ID: "call-1", Name: "mock_tool", Arguments: json.RawMessage(`{"x":1}`)}}},
		{Content: "done"},
	}}

	loop := NewLoop(LoopConfig{LLM: llm, Tools: registry, Scrubber: security.NewScrubber(), MaxTurns: 10})
	execution, err := loop.Run(ctx, "run the tool")
	if err != nil {
		t.Fatalf("loop.Run() error = %v", err)
	}

	if execution.TurnCount != 2 {
		t.Fatalf("TurnCount = %d, want 2", execution.TurnCount)
	}
	if tool.callCount != 1 {
		t.Fatalf("tool callCount = %d, want 1", tool.callCount)
	}
	if execution.Response != "done" {
		t.Fatalf("Response = %q, want %q", execution.Response, "done")
	}
	if len(execution.Messages) != 5 {
		t.Fatalf("len(Messages) = %d, want 5", len(execution.Messages))
	}
	if execution.Messages[0].Role != "system" || execution.Messages[1].Role != "user" || execution.Messages[2].Role != "assistant" || execution.Messages[3].Role != "tool" || execution.Messages[4].Role != "assistant" {
		t.Fatalf("unexpected role sequence: %+v", []string{execution.Messages[0].Role, execution.Messages[1].Role, execution.Messages[2].Role, execution.Messages[3].Role, execution.Messages[4].Role})
	}
}

func TestAgentLoopMaxTurns(t *testing.T) {
	ctx := context.Background()
	registry := tools.NewRegistry()
	registry.Register(&loopMockTool{name: "loop_tool", result: "ok"})

	llm := &loopMockLLM{responses: []types.LLMResponse{{
		ToolCalls: []types.ToolCall{{ID: "call-1", Name: "loop_tool", Arguments: json.RawMessage(`{}`)}},
	}}}

	loop := NewLoop(LoopConfig{LLM: llm, Tools: registry, Scrubber: security.NewScrubber(), MaxTurns: 3})
	execution, err := loop.Run(ctx, "keep going")
	if !errors.Is(err, types.ErrMaxTurnsExceeded) {
		t.Fatalf("error = %v, want ErrMaxTurnsExceeded", err)
	}
	if execution == nil {
		t.Fatal("execution is nil")
	}
	if execution.TurnCount != 3 {
		t.Fatalf("TurnCount = %d, want 3", execution.TurnCount)
	}
}

func TestAgentLoopNoTools(t *testing.T) {
	ctx := context.Background()
	llm := &loopMockLLM{responses: []types.LLMResponse{{Content: "immediate answer"}}}
	loop := NewLoop(LoopConfig{LLM: llm, Tools: tools.NewRegistry(), Scrubber: security.NewScrubber(), MaxTurns: 10})

	execution, err := loop.Run(ctx, "hello")
	if err != nil {
		t.Fatalf("loop.Run() error = %v", err)
	}
	if execution.TurnCount != 1 {
		t.Fatalf("TurnCount = %d, want 1", execution.TurnCount)
	}
	if execution.Response != "immediate answer" {
		t.Fatalf("Response = %q, want %q", execution.Response, "immediate answer")
	}
}

func TestAgentLoopScrubbing(t *testing.T) {
	ctx := context.Background()
	registry := tools.NewRegistry()
	tool := &loopMockTool{name: "secret_tool", result: "token sk-abc123secretkeyABCDEFGHIJKLMNO leaked"}
	registry.Register(tool)

	llm := &loopMockLLM{responses: []types.LLMResponse{
		{ToolCalls: []types.ToolCall{{ID: "call-1", Name: "secret_tool", Arguments: json.RawMessage(`{}`)}}},
		{Content: "done"},
	}}

	loop := NewLoop(LoopConfig{LLM: llm, Tools: registry, Scrubber: security.NewScrubber(), MaxTurns: 10})
	execution, err := loop.Run(ctx, "get secret")
	if err != nil {
		t.Fatalf("loop.Run() error = %v", err)
	}

	toolMessage := execution.Messages[3].Content
	if strings.Contains(toolMessage, "sk-abc123secretkey") {
		t.Fatalf("tool output not scrubbed: %q", toolMessage)
	}
	if !strings.Contains(toolMessage, "[REDACTED]") {
		t.Fatalf("tool output missing redaction marker: %q", toolMessage)
	}
}

func TestExecutionAddMessages(t *testing.T) {
	execution := NewExecution(0)
	execution.AddSystemMessage("sys")
	execution.AddUserMessage("user")
	execution.AddAssistantMessage("assistant", []types.ToolCall{{ID: "call-1", Name: "x"}})
	execution.AddToolResult("call-1", "x", "result")

	if execution.MaxTurns != 50 {
		t.Fatalf("MaxTurns = %d, want 50", execution.MaxTurns)
	}
	if len(execution.Messages) != 4 {
		t.Fatalf("len(Messages) = %d, want 4", len(execution.Messages))
	}
	if execution.ToolOutputs["call-1"] != "result" {
		t.Fatalf("ToolOutputs[call-1] = %q, want %q", execution.ToolOutputs["call-1"], "result")
	}
}

func TestSystemPromptAssembly(t *testing.T) {
	ctx := context.Background()
	workspace := t.TempDir()

	if err := os.WriteFile(filepath.Join(workspace, "AGENTS.md"), []byte("agent rules"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "SOUL.md"), []byte("soul rules"), 0o644); err != nil {
		t.Fatalf("write SOUL.md: %v", err)
	}

	llm := &loopMockLLM{responses: []types.LLMResponse{{Content: "ok"}}}
	loop := NewLoop(LoopConfig{
		LLM:          llm,
		Tools:        tools.NewRegistry(),
		Scrubber:     security.NewScrubber(),
		WorkspaceDir: workspace,
		MaxTurns:     5,
	})

	execution, err := loop.Run(ctx, "hello")
	if err != nil {
		t.Fatalf("loop.Run() error = %v", err)
	}

	system := execution.Messages[0].Content
	if !strings.Contains(system, "agent rules") {
		t.Fatalf("system prompt missing AGENTS.md content: %q", system)
	}
	if !strings.Contains(system, "soul rules") {
		t.Fatalf("system prompt missing SOUL.md content: %q", system)
	}
}

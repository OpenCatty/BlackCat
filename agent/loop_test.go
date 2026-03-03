package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/startower-observability/blackcat/security"
	"github.com/startower-observability/blackcat/tools"
	"github.com/startower-observability/blackcat/types"
)

type toolErrorMockTool struct {
	name string
}

func (m *toolErrorMockTool) Name() string { return m.name }

func (m *toolErrorMockTool) Description() string { return "mock tool that fails" }

func (m *toolErrorMockTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }

func (m *toolErrorMockTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	_ = ctx
	_ = args
	return "", errors.New("boom")
}

func TestToolErrorRecovery(t *testing.T) {
	ctx := context.Background()
	registry := tools.NewRegistry()
	registry.Register(&toolErrorMockTool{name: "failing_tool"})

	llm := &loopMockLLM{responses: []types.LLMResponse{
		{ToolCalls: []types.ToolCall{{ID: "call-1", Name: "failing_tool", Arguments: json.RawMessage(`{"x":1}`)}}},
		{Content: "done"},
	}}

	loop := NewLoop(LoopConfig{LLM: llm, Tools: registry, Scrubber: security.NewScrubber(), MaxTurns: 10})
	execution, err := loop.Run(ctx, "run tool")
	if err != nil {
		t.Fatalf("loop.Run() error = %v", err)
	}
	if execution.Error != nil {
		t.Fatalf("execution.Error = %v, want nil", execution.Error)
	}

	foundToolErrorResult := false
	for _, msg := range execution.Messages {
		if msg.Role == "tool" && msg.Content == "Tool error: boom" {
			foundToolErrorResult = true
			break
		}
	}
	if !foundToolErrorResult {
		t.Fatalf("expected tool result message %q in execution.Messages, got %+v", "Tool error: boom", execution.Messages)
	}
}

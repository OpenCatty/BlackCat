package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/startower-observability/blackcat/internal/hooks"
	"github.com/startower-observability/blackcat/security"
	"github.com/startower-observability/blackcat/tools"
	"github.com/startower-observability/blackcat/types"
)

func TestHooksFireInOrder(t *testing.T) {
	ctx := context.Background()
	registry := tools.NewRegistry()
	tool := &loopMockTool{name: "mock_tool", result: "tool output"}
	registry.Register(tool)

	llm := &loopMockLLM{responses: []types.LLMResponse{
		{ToolCalls: []types.ToolCall{{ID: "call-1", Name: "mock_tool", Arguments: json.RawMessage(`{"x":1}`)}}},
		{Content: "done"},
	}}

	hookRegistry := hooks.NewHookRegistry()
	order := make([]hooks.HookEvent, 0, 6)

	hookRegistry.Register(hooks.PreChat, func(hctx *hooks.HookContext) error {
		order = append(order, hooks.PreChat)
		if hctx.Metadata == nil {
			t.Fatal("PreChat metadata is nil")
		}
		if _, ok := hctx.Metadata["messages"]; !ok {
			t.Fatal("PreChat metadata missing messages")
		}
		return nil
	})

	hookRegistry.Register(hooks.PostChat, func(hctx *hooks.HookContext) error {
		order = append(order, hooks.PostChat)
		if hctx.LLMResponse == nil {
			t.Fatal("PostChat missing LLM response")
		}
		return nil
	})

	hookRegistry.Register(hooks.PreToolExec, func(hctx *hooks.HookContext) error {
		order = append(order, hooks.PreToolExec)
		if hctx.ToolName != "mock_tool" {
			t.Fatalf("PreToolExec ToolName = %q, want %q", hctx.ToolName, "mock_tool")
		}
		if hctx.Metadata == nil {
			t.Fatal("PreToolExec metadata is nil")
		}
		if _, ok := hctx.Metadata["args"]; !ok {
			t.Fatal("PreToolExec metadata missing args")
		}
		return nil
	})

	hookRegistry.Register(hooks.PostToolExec, func(hctx *hooks.HookContext) error {
		order = append(order, hooks.PostToolExec)
		if hctx.Metadata == nil {
			t.Fatal("PostToolExec metadata is nil")
		}
		if _, ok := hctx.Metadata["result"]; !ok {
			t.Fatal("PostToolExec metadata missing result")
		}
		return nil
	})

	loop := NewLoop(LoopConfig{
		LLM:      llm,
		Tools:    registry,
		Scrubber: security.NewScrubber(),
		Hooks:    hookRegistry,
		MaxTurns: 10,
	})

	execution, err := loop.Run(ctx, "run the tool")
	if err != nil {
		t.Fatalf("loop.Run() error = %v", err)
	}

	if execution.Response != "done" {
		t.Fatalf("Response = %q, want %q", execution.Response, "done")
	}

	if len(order) < 4 {
		t.Fatalf("hook call count = %d, want at least 4", len(order))
	}

	wantPrefix := []hooks.HookEvent{hooks.PreChat, hooks.PostChat, hooks.PreToolExec, hooks.PostToolExec}
	for i := range wantPrefix {
		if order[i] != wantPrefix[i] {
			t.Fatalf("hook order[%d] = %q, want %q (full: %v)", i, order[i], wantPrefix[i], order)
		}
	}
}

func TestNilHooksNoChange(t *testing.T) {
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
}

func TestPreToolExecErrorSkipsTool(t *testing.T) {
	ctx := context.Background()
	registry := tools.NewRegistry()
	tool := &loopMockTool{name: "mock_tool", result: "tool output"}
	registry.Register(tool)

	llm := &loopMockLLM{responses: []types.LLMResponse{
		{ToolCalls: []types.ToolCall{{ID: "call-1", Name: "mock_tool", Arguments: json.RawMessage(`{"x":1}`)}}},
		{Content: "done"},
	}}

	hookRegistry := hooks.NewHookRegistry()
	hookRegistry.Register(hooks.PreToolExec, func(hctx *hooks.HookContext) error {
		return context.Canceled
	})

	loop := NewLoop(LoopConfig{
		LLM:      llm,
		Tools:    registry,
		Scrubber: security.NewScrubber(),
		Hooks:    hookRegistry,
		MaxTurns: 10,
	})

	execution, err := loop.Run(ctx, "run the tool")
	if err != nil {
		t.Fatalf("loop.Run() error = %v", err)
	}

	if tool.callCount != 0 {
		t.Fatalf("tool callCount = %d, want 0", tool.callCount)
	}
	if execution.Response != "done" {
		t.Fatalf("Response = %q, want %q", execution.Response, "done")
	}
	if len(execution.Messages) != 4 {
		t.Fatalf("len(Messages) = %d, want 4", len(execution.Messages))
	}
	if execution.Messages[2].Role != "assistant" || execution.Messages[3].Role != "assistant" {
		t.Fatalf("unexpected role sequence after skipped tool: %q, %q", execution.Messages[2].Role, execution.Messages[3].Role)
	}
}

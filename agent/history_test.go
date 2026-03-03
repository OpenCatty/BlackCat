package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/startower-observability/blackcat/security"
	"github.com/startower-observability/blackcat/tools"
	"github.com/startower-observability/blackcat/types"
)

type historyCaptureLLM struct {
	responses    []types.LLMResponse
	callCount    int
	lastMessages []types.LLMMessage
}

func (m *historyCaptureLLM) Chat(ctx context.Context, messages []types.LLMMessage, toolDefs []types.ToolDefinition) (*types.LLMResponse, error) {
	_ = ctx
	_ = toolDefs

	m.callCount++
	m.lastMessages = append([]types.LLMMessage(nil), messages...)

	if len(m.responses) == 0 {
		return &types.LLMResponse{}, nil
	}

	idx := m.callCount - 1
	if idx >= len(m.responses) {
		idx = len(m.responses) - 1
	}

	resp := m.responses[idx]
	return &resp, nil
}

func (m *historyCaptureLLM) Stream(ctx context.Context, messages []types.LLMMessage, toolDefs []types.ToolDefinition) (<-chan types.Chunk, error) {
	_ = ctx
	_ = messages
	_ = toolDefs

	ch := make(chan types.Chunk)
	close(ch)
	return ch, nil
}

func TestRunWithHistory(t *testing.T) {
	llm := &historyCaptureLLM{responses: []types.LLMResponse{{Content: "ok"}}}
	history := []types.LLMMessage{
		{Role: "user", Content: "old user 1"},
		{Role: "tool", Content: "old tool 1"},
		{Role: "assistant", Content: "old assistant 1"},
		{Role: "system", Content: "old system 1"},
		{Role: "user", Content: "old user 2"},
	}

	expectedInjected := []types.LLMMessage{
		{Role: "user", Content: "old user 1"},
		{Role: "assistant", Content: "old assistant 1"},
		{Role: "user", Content: "old user 2"},
	}

	loop := NewLoop(LoopConfig{
		LLM:             llm,
		Tools:           tools.NewRegistry(),
		Scrubber:        security.NewScrubber(),
		SessionMessages: history,
	})

	if _, err := loop.Run(context.Background(), "current user message"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(llm.lastMessages) != 5 {
		t.Fatalf("len(lastMessages) = %d, want 5", len(llm.lastMessages))
	}

	if llm.lastMessages[0].Role != "system" {
		t.Fatalf("lastMessages[0].Role = %q, want system", llm.lastMessages[0].Role)
	}

	for i := range expectedInjected {
		injected := llm.lastMessages[i+1]
		if injected.Role != expectedInjected[i].Role || injected.Content != expectedInjected[i].Content {
			t.Fatalf("history message %d mismatch: got %#v want %#v", i, injected, expectedInjected[i])
		}
	}

	if llm.lastMessages[4].Role != "user" || llm.lastMessages[4].Content != "current user message" {
		t.Fatalf("current user message mismatch: got %#v", llm.lastMessages[4])
	}
}

func TestHistoryLimit(t *testing.T) {
	llm := &historyCaptureLLM{responses: []types.LLMResponse{{Content: "ok"}}}
	history := make([]types.LLMMessage, 0, 50)
	for i := 0; i < 50; i++ {
		role := "assistant"
		if i%2 == 0 {
			role = "user"
		}

		history = append(history, types.LLMMessage{
			Role:    role,
			Content: fmt.Sprintf("history-%02d", i),
		})
	}

	loop := NewLoop(LoopConfig{
		LLM:                llm,
		Tools:              tools.NewRegistry(),
		Scrubber:           security.NewScrubber(),
		SessionMessages:    history,
		MaxHistoryMessages: 20,
	})

	if _, err := loop.Run(context.Background(), "current user message"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(llm.lastMessages) != 22 {
		t.Fatalf("len(lastMessages) = %d, want 22", len(llm.lastMessages))
	}

	expected := history[30:]
	for i := range expected {
		injected := llm.lastMessages[i+1]
		if injected.Role != expected[i].Role || injected.Content != expected[i].Content {
			t.Fatalf("limited history message %d mismatch: got %#v want %#v", i, injected, expected[i])
		}
	}

	if llm.lastMessages[21].Role != "user" || llm.lastMessages[21].Content != "current user message" {
		t.Fatalf("current user message mismatch: got %#v", llm.lastMessages[21])
	}
}

func TestRunNoHistory(t *testing.T) {
	testCases := []struct {
		name    string
		history []types.LLMMessage
	}{
		{name: "nil history", history: nil},
		{name: "empty history", history: []types.LLMMessage{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			llm := &historyCaptureLLM{responses: []types.LLMResponse{{Content: "ok"}}}
			loop := NewLoop(LoopConfig{
				LLM:             llm,
				Tools:           tools.NewRegistry(),
				Scrubber:        security.NewScrubber(),
				SessionMessages: tc.history,
			})

			if _, err := loop.Run(context.Background(), "current user message"); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if len(llm.lastMessages) != 2 {
				t.Fatalf("len(lastMessages) = %d, want 2", len(llm.lastMessages))
			}

			if llm.lastMessages[0].Role != "system" {
				t.Fatalf("lastMessages[0].Role = %q, want system", llm.lastMessages[0].Role)
			}

			if llm.lastMessages[1].Role != "user" || llm.lastMessages[1].Content != "current user message" {
				t.Fatalf("current user message mismatch: got %#v", llm.lastMessages[1])
			}
		})
	}
}

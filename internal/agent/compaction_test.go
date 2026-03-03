package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/startower-observability/blackcat/internal/types"
)

type mockCompactionLLM struct {
	responses []types.LLMResponse
	callIndex int
}

func (m *mockCompactionLLM) Chat(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (*types.LLMResponse, error) {
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

func (m *mockCompactionLLM) Stream(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (<-chan types.Chunk, error) {
	_ = ctx
	_ = messages
	_ = tools
	ch := make(chan types.Chunk)
	close(ch)
	return ch, nil
}

func TestShouldCompactBelowThreshold(t *testing.T) {
	compactor := NewCompactor(CompactorConfig{
		MaxTokens:   128000,
		Threshold:   0.835,
		MinMessages: 1,
	})

	messages := []types.LLMMessage{
		{Role: "user", Content: "short one"},
		{Role: "assistant", Content: "short two"},
		{Role: "user", Content: "short three"},
		{Role: "assistant", Content: "short four"},
		{Role: "user", Content: "short five"},
	}

	if compactor.ShouldCompact(messages) {
		t.Fatalf("ShouldCompact() = true, want false")
	}
}

func TestShouldCompactAboveThreshold(t *testing.T) {
	compactor := NewCompactor(CompactorConfig{
		MaxTokens:   128000,
		Threshold:   0.835,
		MinMessages: 1,
	})

	long := strings.Repeat("a", 430000)
	messages := []types.LLMMessage{
		{Role: "user", Content: long},
		{Role: "assistant", Content: long},
	}

	if !compactor.ShouldCompact(messages) {
		t.Fatalf("ShouldCompact() = false, want true")
	}
}

func TestShouldCompactMinMessages(t *testing.T) {
	compactor := NewCompactor(CompactorConfig{
		MaxTokens:   128000,
		Threshold:   0.835,
		MinMessages: 3,
	})

	long := strings.Repeat("b", 430000)
	messages := []types.LLMMessage{
		{Role: "user", Content: long},
		{Role: "assistant", Content: long},
		{Role: "user", Content: long},
	}

	if compactor.ShouldCompact(messages) {
		t.Fatalf("ShouldCompact() = true with message count <= minMessages, want false")
	}
}

func TestCompactPreservesSystem(t *testing.T) {
	llm := &mockCompactionLLM{responses: []types.LLMResponse{{Content: "Summary of conversation"}}}
	compactor := NewCompactor(CompactorConfig{LLM: llm, MinMessages: 3})

	messages := []types.LLMMessage{
		{Role: "system", Content: "You are a coding assistant."},
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "second"},
		{Role: "user", Content: "third"},
		{Role: "assistant", Content: "fourth"},
		{Role: "user", Content: "fifth"},
		{Role: "assistant", Content: "sixth"},
	}

	got, err := compactor.Compact(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}

	if len(got) == 0 || got[0].Role != "system" || got[0].Content != messages[0].Content {
		t.Fatalf("system message not preserved: got %#v", got)
	}
}

func TestCompactPreservesRecentMessages(t *testing.T) {
	llm := &mockCompactionLLM{responses: []types.LLMResponse{{Content: "Summary of conversation"}}}
	compactor := NewCompactor(CompactorConfig{LLM: llm, MinMessages: 3})

	messages := []types.LLMMessage{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "m1"},
		{Role: "assistant", Content: "m2"},
		{Role: "user", Content: "m3"},
		{Role: "assistant", Content: "m4"},
		{Role: "user", Content: "m5"},
		{Role: "assistant", Content: "m6"},
	}

	got, err := compactor.Compact(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}

	if len(got) < 5 {
		t.Fatalf("Compact() returned too few messages: %d", len(got))
	}

	last := got[len(got)-3:]
	for i := range last {
		want := messages[len(messages)-3+i]
		if last[i].Role != want.Role || last[i].Content != want.Content {
			t.Fatalf("recent message %d mismatch: got %#v want %#v", i, last[i], want)
		}
	}
}

func TestCompactSummarizes(t *testing.T) {
	llm := &mockCompactionLLM{responses: []types.LLMResponse{{Content: "Summary of conversation"}}}
	compactor := NewCompactor(CompactorConfig{LLM: llm, MinMessages: 2})

	messages := []types.LLMMessage{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "m1"},
		{Role: "assistant", Content: "m2"},
		{Role: "user", Content: "m3"},
		{Role: "assistant", Content: "m4"},
	}

	got, err := compactor.Compact(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}

	if len(got) < 2 {
		t.Fatalf("Compact() returned too few messages: %d", len(got))
	}

	summary := got[1]
	if summary.Role != "assistant" {
		t.Fatalf("summary role = %q, want assistant", summary.Role)
	}
	if !strings.HasPrefix(summary.Content, "[Compaction Summary]\n") {
		t.Fatalf("summary missing prefix: %q", summary.Content)
	}
	if !strings.Contains(summary.Content, "<!-- compaction-boundary -->") {
		t.Fatalf("summary missing compaction boundary tag: %q", summary.Content)
	}
	if !strings.Contains(summary.Content, "Summary of conversation") {
		t.Fatalf("summary missing llm output: %q", summary.Content)
	}
}

func TestCompactNothingToCompact(t *testing.T) {
	llm := &mockCompactionLLM{responses: []types.LLMResponse{{Content: "Summary of conversation"}}}
	compactor := NewCompactor(CompactorConfig{LLM: llm, MinMessages: 3})

	messages := []types.LLMMessage{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "m1"},
		{Role: "assistant", Content: "m2"},
		{Role: "user", Content: "m3"},
	}

	got, err := compactor.Compact(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}

	if len(got) != len(messages) {
		t.Fatalf("Compact() length = %d, want %d", len(got), len(messages))
	}
	for i := range messages {
		if got[i].Role != messages[i].Role || got[i].Content != messages[i].Content {
			t.Fatalf("Compact() changed message %d: got %#v want %#v", i, got[i], messages[i])
		}
	}
	if llm.callIndex != 0 {
		t.Fatalf("expected llm not to be called, called %d times", llm.callIndex)
	}
}

func TestEstimateTokens(t *testing.T) {
	content := strings.Repeat("x", 40)
	args := json.RawMessage(`{"command":"build","path":"./agent"}`)
	messages := []types.LLMMessage{
		{
			Role:    "user",
			Content: content,
			ToolCalls: []types.ToolCall{
				{Name: "exec", Arguments: args},
			},
		},
	}

	got := estimateTokens(messages)
	want := len(content)/4 + len(args)/4
	if got != want {
		t.Fatalf("estimateTokens() = %d, want %d", got, want)
	}
}

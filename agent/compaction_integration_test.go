package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/startower-observability/blackcat/types"
)

// failingMockLLM is a mock LLM that always fails
type failingMockLLM struct{}

func (f *failingMockLLM) Chat(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (*types.LLMResponse, error) {
	return nil, fmt.Errorf("mock llm failure for testing")
}

func (f *failingMockLLM) Stream(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (<-chan types.Chunk, error) {
	ch := make(chan types.Chunk)
	close(ch)
	return ch, fmt.Errorf("mock llm failure for testing")
}

// TestLoopRunProactiveCompaction verifies that execution.Compacted == true
// when context exceeds threshold
func TestLoopRunProactiveCompaction(t *testing.T) {
	// Setup: Create a loop with very low maxContextTokens so ShouldCompact triggers
	// ShouldCompact threshold: 0.80 * 100 = 80 tokens
	// estimateTokens: len(content)/4 per message
	// So need messages with total content > 320 chars to exceed 80 tokens
	// AND need > 6 messages (minMessages)

	mockLLM := &mockCompactionLLM{
		responses: []types.LLMResponse{
			// First response: for Compact() summary
			{
				Content: "Summary of past conversation",
				Usage: types.LLMUsage{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:      15,
				},
			},
			// Second response: for main Run() call
			{
				Content: "ok",
				Usage: types.LLMUsage{
					PromptTokens:     10,
					CompletionTokens: 2,
					TotalTokens:      12,
				},
			},
		},
	}

	loop := NewLoop(LoopConfig{
		LLM:              mockLLM,
		MaxContextTokens: 100,
		MaxTurns:         5,
		SessionMessages: []types.LLMMessage{
			// 7 messages to exceed minMessages=6
			// Each ~50 chars to exceed 320 total (80*4)
			{Role: "user", Content: "message one message one m"},
			{Role: "assistant", Content: "response one response one"},
			{Role: "user", Content: "message two message two m"},
			{Role: "assistant", Content: "response two response two"},
			{Role: "user", Content: "message three message thm"},
			{Role: "assistant", Content: "response three response t"},
			{Role: "user", Content: "message four message four"},
		},
	})

	execution, err := loop.Run(context.Background(), "test message")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !execution.Compacted {
		t.Fatalf("execution.Compacted = false, want true")
	}

	if execution.Response != "ok" {
		t.Fatalf("execution.Response = %q, want %q", execution.Response, "ok")
	}
}

// TestLoopRunNoCompactionWhenUnderThreshold verifies that execution.Compacted == false
// when context is small
func TestLoopRunNoCompactionWhenUnderThreshold(t *testing.T) {
	mockLLM := &mockCompactionLLM{
		responses: []types.LLMResponse{
			{
				Content: "Short answer",
				Usage: types.LLMUsage{
					PromptTokens:     5,
					CompletionTokens: 2,
					TotalTokens:      7,
				},
			},
		},
	}

	loop := NewLoop(LoopConfig{
		LLM:              mockLLM,
		MaxContextTokens: 80000, // Normal high limit
		MaxTurns:         5,
		SessionMessages: []types.LLMMessage{
			// Only 2 short messages - well under threshold
			{Role: "user", Content: "short"},
			{Role: "assistant", Content: "ok"},
		},
	})

	execution, err := loop.Run(context.Background(), "test message")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if execution.Compacted {
		t.Fatalf("execution.Compacted = true, want false")
	}

	if execution.Response != "Short answer" {
		t.Fatalf("execution.Response = %q, want %q", execution.Response, "Short answer")
	}
}

// TestLoopRunCompactionFailureStillRuns verifies that loop still runs
// even when compaction LLM call fails
func TestLoopRunCompactionFailureStillRuns(t *testing.T) {
	// Create a mock that returns an error on first call (compaction)
	// and succeeds on second call (main LLM)
	mockLLM := &mockCompactionLLM{
		responses: []types.LLMResponse{
			// This will be returned for the main chat after compaction fails
			{
				Content: "Fallback response",
				Usage: types.LLMUsage{
					PromptTokens:     10,
					CompletionTokens: 3,
					TotalTokens:      13,
				},
			},
		},
	}

	// Wrap it to fail on first call, succeed on second
	failFirstThenSucceedLLM := &selectiveFailLLM{
		underlying: mockLLM,
		failFirst:  true,
	}

	loop := NewLoop(LoopConfig{
		LLM:              failFirstThenSucceedLLM,
		MaxContextTokens: 100,
		MaxTurns:         5,
		SessionMessages: []types.LLMMessage{
			// 7 messages to trigger ShouldCompact
			{Role: "user", Content: "message one message one m"},
			{Role: "assistant", Content: "response one response one"},
			{Role: "user", Content: "message two message two m"},
			{Role: "assistant", Content: "response two response two"},
			{Role: "user", Content: "message three message thm"},
			{Role: "assistant", Content: "response three response t"},
			{Role: "user", Content: "message four message four"},
		},
	})

	execution, err := loop.Run(context.Background(), "test message")
	if err != nil {
		t.Fatalf("Run() error = %v, expected graceful degradation", err)
	}

	if execution == nil {
		t.Fatalf("execution = nil, want non-nil")
	}

	// Compaction failed, so Compacted should be false
	if execution.Compacted {
		t.Fatalf("execution.Compacted = true, want false (compaction should have failed)")
	}

	// But main response should still be present (graceful degradation)
	if execution.Response != "Fallback response" {
		t.Fatalf("execution.Response = %q, want %q", execution.Response, "Fallback response")
	}

	// Loop should have completed
	if !execution.Done {
		t.Fatalf("execution.Done = false, want true")
	}
}

// selectiveFailLLM wraps another LLM and fails on the first call
type selectiveFailLLM struct {
	underlying *mockCompactionLLM
	failFirst  bool
	callCount  int
}

func (s *selectiveFailLLM) Chat(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (*types.LLMResponse, error) {
	if s.failFirst && s.callCount == 0 {
		s.callCount++
		return nil, fmt.Errorf("simulated compaction llm failure")
	}
	s.callCount++
	return s.underlying.Chat(ctx, messages, tools)
}

func (s *selectiveFailLLM) Stream(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (<-chan types.Chunk, error) {
	return s.underlying.Stream(ctx, messages, tools)
}

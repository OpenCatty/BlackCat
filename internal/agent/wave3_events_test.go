package agent

import (
	"context"
	"testing"

	"github.com/startower-observability/blackcat/internal/security"
	"github.com/startower-observability/blackcat/internal/types"
)

// TestEmit_NilEventStream verifies that calling emit with a nil EventStream
// does not panic. This is the default case when no event consumer is attached.
func TestEmit_NilEventStream(t *testing.T) {
	loop := &Loop{} // eventStream is nil by default

	// Should not panic
	loop.emit(nil, AgentEvent{Kind: EventThinking, TurnNum: 1, Message: "test"})
	loop.emit(nil, AgentEvent{Kind: EventDone, TurnNum: 1, Result: "done"})
	loop.emit(nil, AgentEvent{Kind: EventError, TurnNum: 1, Error: "oops"})
	loop.emit(nil, AgentEvent{Kind: EventToolCallStart, TurnNum: 1, ToolName: "test_tool"})
	loop.emit(nil, AgentEvent{Kind: EventToolCallResult, TurnNum: 1, ToolName: "test_tool", Result: "ok"})
}

// TestLoop_NilEventStream_RunDoesNotPanic runs the full agent loop with a nil
// EventStream to verify emit calls inside processOneTurn don't panic.
func TestLoop_NilEventStream_RunDoesNotPanic(t *testing.T) {
	llm := &loopMockLLM{responses: []types.LLMResponse{
		{Content: "hello"},
	}}

	loop := NewLoop(LoopConfig{
		LLM:      llm,
		Scrubber: security.NewScrubber(),
		MaxTurns: 5,
		// EventStream deliberately nil
	})

	ctx := context.Background()
	exec, err := loop.Run(ctx, "test message")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if exec.Response != "hello" {
		t.Errorf("Response = %q, want %q", exec.Response, "hello")
	}
}

// TestLoop_WithEventStream_ReceivesEvents verifies that events are delivered
// to a non-nil EventStream channel.
func TestLoop_WithEventStream_ReceivesEvents(t *testing.T) {
	llm := &loopMockLLM{responses: []types.LLMResponse{
		{Content: "done"},
	}}

	eventCh := make(chan AgentEvent, 16)

	loop := NewLoop(LoopConfig{
		LLM:         llm,
		Scrubber:    security.NewScrubber(),
		MaxTurns:    5,
		EventStream: eventCh,
	})

	ctx := context.Background()
	_, err := loop.Run(ctx, "hi")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should have received at least a Thinking and Done event
	close(eventCh)
	var kinds []EventKind
	for ev := range eventCh {
		kinds = append(kinds, ev.Kind)
	}

	if len(kinds) == 0 {
		t.Fatal("expected at least one event, got none")
	}

	hasThinking := false
	hasDone := false
	for _, k := range kinds {
		if k == EventThinking {
			hasThinking = true
		}
		if k == EventDone {
			hasDone = true
		}
	}
	if !hasThinking {
		t.Error("expected EventThinking event")
	}
	if !hasDone {
		t.Error("expected EventDone event")
	}
}

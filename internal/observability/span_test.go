package observability

import (
	"context"
	"testing"
	"time"
)

func TestNewTurnSpan_Fields(t *testing.T) {
	traceID := "abc123def456"
	turnNum := 3
	model := "gpt-4"
	provider := "openai"

	span := NewTurnSpan(traceID, turnNum, model, provider)

	if span.TraceID != traceID {
		t.Errorf("TraceID = %q, want %q", span.TraceID, traceID)
	}
	if span.TurnNum != turnNum {
		t.Errorf("TurnNum = %d, want %d", span.TurnNum, turnNum)
	}
	if span.Model != model {
		t.Errorf("Model = %q, want %q", span.Model, model)
	}
	if span.Provider != provider {
		t.Errorf("Provider = %q, want %q", span.Provider, provider)
	}
	if span.StartedAt.IsZero() {
		t.Error("StartedAt should not be zero")
	}
	if !span.EndedAt.IsZero() {
		t.Error("EndedAt should be zero before End() is called")
	}
	if span.InputTokens != 0 {
		t.Errorf("InputTokens = %d, want 0", span.InputTokens)
	}
	if span.OutputTokens != 0 {
		t.Errorf("OutputTokens = %d, want 0", span.OutputTokens)
	}
}

func TestTurnSpan_Duration(t *testing.T) {
	span := NewTurnSpan("trace1", 1, "model", "provider")

	// Duration before End should still work (uses time.Since)
	preDur := span.Duration()
	if preDur < 0 {
		t.Errorf("Duration before End() = %v, want >= 0", preDur)
	}

	// Small sleep to ensure measurable duration
	time.Sleep(5 * time.Millisecond)

	span.End(context.Background())

	dur := span.Duration()
	if dur < 5*time.Millisecond {
		t.Errorf("Duration after End() = %v, want >= 5ms", dur)
	}
	if span.EndedAt.IsZero() {
		t.Error("EndedAt should not be zero after End()")
	}
}

func TestTurnSpan_Outcome(t *testing.T) {
	outcomes := []string{"run_again", "final_output", "interrupted", "handoff", "error"}

	for _, outcome := range outcomes {
		t.Run(outcome, func(t *testing.T) {
			span := NewTurnSpan("trace1", 1, "model", "provider")
			span.Outcome = outcome
			span.InputTokens = 100
			span.OutputTokens = 50
			span.ToolCallCount = 2
			span.ToolNames = []string{"read_file", "write_file"}

			if outcome == "error" {
				span.ErrorMsg = "something went wrong"
			}

			// Should not panic
			span.End(context.Background())

			if span.Outcome != outcome {
				t.Errorf("Outcome = %q, want %q", span.Outcome, outcome)
			}
		})
	}
}

func TestNewTraceID_Unique(t *testing.T) {
	id1 := NewTraceID()
	id2 := NewTraceID()

	if len(id1) != 16 {
		t.Errorf("TraceID length = %d, want 16", len(id1))
	}
	if len(id2) != 16 {
		t.Errorf("TraceID length = %d, want 16", len(id2))
	}
	if id1 == id2 {
		t.Errorf("Two TraceIDs should be unique, both are %q", id1)
	}

	// Verify hex encoding (all chars should be [0-9a-f])
	for _, c := range id1 {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("TraceID contains non-hex char: %c", c)
		}
	}
}

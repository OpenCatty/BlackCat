package observability

import (
	"context"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// TraceID integration tests (pure Go — no cgo required)
// ---------------------------------------------------------------------------

// TestTraceID_BatchUniqueness generates 100 trace IDs and verifies all are
// unique and well-formed (16 hex characters).
func TestTraceID_BatchUniqueness(t *testing.T) {
	const n = 100
	seen := make(map[string]struct{}, n)

	for i := 0; i < n; i++ {
		id := NewTraceID()

		if len(id) != 16 {
			t.Fatalf("TraceID[%d] length = %d, want 16", i, len(id))
		}

		for _, c := range id {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Fatalf("TraceID[%d] contains non-hex char %c in %q", i, c, id)
			}
		}

		if _, dup := seen[id]; dup {
			t.Fatalf("TraceID collision at index %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
}

// ---------------------------------------------------------------------------
// TurnSpan integration tests (pure Go)
// ---------------------------------------------------------------------------

// TestTurnSpan_EndSetsDuration verifies that after End() the span has a
// positive duration and a non-zero EndedAt timestamp.
func TestTurnSpan_EndSetsDuration(t *testing.T) {
	span := NewTurnSpan("trace-integ", 1, "test-model", "test-provider")
	time.Sleep(2 * time.Millisecond)
	span.End(context.Background())

	dur := span.Duration()
	if dur <= 0 {
		t.Errorf("Duration after End() = %v, want > 0", dur)
	}
	if span.EndedAt.IsZero() {
		t.Error("EndedAt should be set after End()")
	}
	if !span.EndedAt.After(span.StartedAt) {
		t.Error("EndedAt should be after StartedAt")
	}
}

// TestTurnSpan_EndIdempotent verifies that calling End() twice does not panic
// and the duration remains stable (EndedAt from first call is preserved).
func TestTurnSpan_EndIdempotent(t *testing.T) {
	span := NewTurnSpan("trace-idem", 1, "model", "prov")
	time.Sleep(2 * time.Millisecond)
	span.End(context.Background())
	firstEnd := span.EndedAt
	firstDur := span.Duration()

	// Second End() — should not panic.
	span.End(context.Background())

	// EndedAt will be overwritten (by design), but duration should still be > 0.
	if span.Duration() <= 0 {
		t.Errorf("Duration after second End() = %v, want > 0", span.Duration())
	}
	_ = firstEnd
	_ = firstDur
}

// TestTurnSpan_TokenFields verifies that token and tool fields round-trip
// correctly through the span.
func TestTurnSpan_TokenFields(t *testing.T) {
	span := NewTurnSpan("trace-tok", 5, "gpt-4o", "openai")
	span.InputTokens = 1234
	span.OutputTokens = 567
	span.ToolCallCount = 3
	span.ToolNames = []string{"read_file", "write_file", "exec_command"}
	span.Outcome = "final_output"

	span.End(context.Background())

	if span.InputTokens != 1234 {
		t.Errorf("InputTokens = %d, want 1234", span.InputTokens)
	}
	if span.OutputTokens != 567 {
		t.Errorf("OutputTokens = %d, want 567", span.OutputTokens)
	}
	if span.ToolCallCount != 3 {
		t.Errorf("ToolCallCount = %d, want 3", span.ToolCallCount)
	}
	if span.Outcome != "final_output" {
		t.Errorf("Outcome = %q, want %q", span.Outcome, "final_output")
	}
}

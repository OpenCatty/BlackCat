package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"strings"
	"time"
)

// TurnSpan represents a single agent loop iteration with timing and token info.
type TurnSpan struct {
	TraceID  string // correlates all turns in one Run() call
	TurnNum  int
	Model    string
	Provider string

	StartedAt time.Time
	EndedAt   time.Time

	// Populated during the turn
	InputTokens   int
	OutputTokens  int
	ToolCallCount int
	ToolNames     []string
	Outcome       string // "run_again", "final_output", "interrupted", "handoff", "error"
	ErrorMsg      string
}

// NewTurnSpan creates a TurnSpan and records StartedAt.
func NewTurnSpan(traceID string, turnNum int, model, provider string) *TurnSpan {
	return &TurnSpan{
		TraceID:   traceID,
		TurnNum:   turnNum,
		Model:     model,
		Provider:  provider,
		StartedAt: time.Now(),
	}
}

// End records EndedAt and logs the span via slog.Info with all fields as attributes.
func (s *TurnSpan) End(ctx context.Context) {
	s.EndedAt = time.Now()

	attrs := []slog.Attr{
		slog.String("trace_id", s.TraceID),
		slog.Int("turn", s.TurnNum),
		slog.String("model", s.Model),
		slog.String("provider", s.Provider),
		slog.Int64("duration_ms", s.Duration().Milliseconds()),
		slog.Int("input_tokens", s.InputTokens),
		slog.Int("output_tokens", s.OutputTokens),
		slog.Int("tool_calls", s.ToolCallCount),
		slog.String("tool_names", strings.Join(s.ToolNames, ",")),
		slog.String("outcome", s.Outcome),
	}

	if s.ErrorMsg != "" {
		attrs = append(attrs, slog.String("error", s.ErrorMsg))
	}

	args := make([]any, len(attrs))
	for i, a := range attrs {
		args[i] = a
	}

	slog.InfoContext(ctx, "agent.turn", args...)
}

// Duration returns the elapsed time.
func (s *TurnSpan) Duration() time.Duration {
	if s.EndedAt.IsZero() {
		return time.Since(s.StartedAt)
	}
	return s.EndedAt.Sub(s.StartedAt)
}

// NewTraceID generates a short random trace ID for correlating turns in one Run().
// Returns a 16-character hex string from 8 random bytes.
func NewTraceID() string {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback: use timestamp-based ID if crypto/rand fails (extremely unlikely)
		return hex.EncodeToString([]byte(time.Now().Format("20060102")))
	}
	return hex.EncodeToString(b)
}

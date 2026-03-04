package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/startower-observability/blackcat/internal/types"
)

// MaxReflections is the maximum number of self-reflection passes per task run.
const MaxReflections = 2

// maxLessonLength is the maximum character length for a reflection lesson.
const maxLessonLength = 500

// ArchivalStoreIface is a minimal interface for inserting archival memories.
// This avoids importing *memory.SQLiteStore directly and prevents circular imports.
// *memory.SQLiteStore already satisfies this interface.
type ArchivalStoreIface interface {
	InsertArchival(ctx context.Context, userID, content string, tags []string, embedding []float32) error
}

// Reflector generates self-reflection critiques after tool failures and stores
// the lessons learned in archival memory.
type Reflector struct {
	llm   types.LLMClient
	store ArchivalStoreIface
}

// NewReflector creates a new Reflector. If llm or store is nil, the Reflector
// will gracefully no-op on Reflect calls.
func NewReflector(llm types.LLMClient, store ArchivalStoreIface) *Reflector {
	return &Reflector{llm: llm, store: store}
}

// Reflect generates a critique of a tool failure by asking the LLM, then stores
// the lesson in archival memory. Returns the lesson string and any error.
// Gracefully returns ("", nil) if llm or store is nil.
func (r *Reflector) Reflect(ctx context.Context, userID, toolName, toolArgs, toolResult string) (string, error) {
	if r.llm == nil || r.store == nil {
		return "", nil
	}

	prompt := fmt.Sprintf(
		`A tool call just failed. Analyze what went wrong and provide a concise lesson (1-2 sentences) that would help avoid this failure in the future.

Tool: %s
Arguments: %s
Result: %s

Respond with ONLY the lesson learned, no preamble.`, toolName, truncateReflection(toolArgs, 300), truncateReflection(toolResult, 300))

	resp, err := r.llm.Chat(ctx, []types.LLMMessage{
		{Role: "user", Content: prompt},
	}, nil)
	if err != nil {
		slog.WarnContext(ctx, "reflection LLM call failed", "err", err)
		return "", fmt.Errorf("reflection: llm chat: %w", err)
	}

	lesson := resp.Content
	if len(lesson) > maxLessonLength {
		lesson = lesson[:maxLessonLength]
	}

	if lesson == "" {
		return "", nil
	}

	tags := []string{"reflection", toolName}
	if err := r.store.InsertArchival(ctx, userID, lesson, tags, nil); err != nil {
		slog.WarnContext(ctx, "reflection archival insert failed", "err", err)
		return lesson, fmt.Errorf("reflection: archival insert: %w", err)
	}

	return lesson, nil
}

// truncateReflection cuts a string to n chars for the reflection prompt.
func truncateReflection(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

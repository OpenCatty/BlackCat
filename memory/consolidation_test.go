package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/startower-observability/blackcat/types"
)

// mockLLMClient implements types.LLMClient for testing.
type mockLLMClient struct {
	response string
	err      error
}

func (m *mockLLMClient) Chat(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (*types.LLMResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &types.LLMResponse{
		Content: m.response,
		Model:   "test-model",
	}, nil
}

func (m *mockLLMClient) Stream(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (<-chan types.Chunk, error) {
	return nil, fmt.Errorf("stream not implemented in mock")
}

// writeNEntries is a helper that writes n entries to a FileStore and returns the store.
func writeNEntries(t *testing.T, dir string, n int) *FileStore {
	t.Helper()
	fs := NewFileStore(filepath.Join(dir, "MEMORY.md"))
	ctx := context.Background()
	for i := 0; i < n; i++ {
		err := fs.Write(ctx, Entry{
			Timestamp: time.Date(2025, 6, 1, i%24, 0, 0, 0, time.UTC),
			Content:   fmt.Sprintf("Memory entry number %d with some context.", i+1),
			Tags:      []string{"test", fmt.Sprintf("entry-%d", i+1)},
		})
		if err != nil {
			t.Fatalf("writeNEntries: write %d failed: %v", i, err)
		}
	}
	return fs
}

// buildConsolidatedResponse builds a valid markdown response with fewer entries.
func buildConsolidatedResponse(n int) string {
	var buf strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			buf.WriteByte('\n')
		}
		ts := time.Date(2025, 6, 1, i%24, 0, 0, 0, time.UTC).UTC().Format(time.RFC3339)
		buf.WriteString("## " + ts + "\n")
		buf.WriteString("**Tags**: consolidated\n")
		buf.WriteString(fmt.Sprintf("Consolidated summary entry %d.\n", i+1))
	}
	return buf.String()
}

func TestConsolidateWithLLMBelowThreshold(t *testing.T) {
	dir := t.TempDir()
	fs := writeNEntries(t, dir, 5)
	ctx := context.Background()

	mock := &mockLLMClient{response: "should not be called"}

	err := fs.ConsolidateWithLLM(ctx, mock, 50)
	if err != nil {
		t.Fatalf("ConsolidateWithLLM below threshold should return nil, got: %v", err)
	}

	// Verify no backup was created.
	bakPath := filepath.Join(dir, "MEMORY.md.bak")
	if _, err := os.Stat(bakPath); !os.IsNotExist(err) {
		t.Fatalf("backup file should not exist below threshold, stat err: %v", err)
	}
}

func TestConsolidateWithLLMAboveThreshold(t *testing.T) {
	dir := t.TempDir()
	fs := writeNEntries(t, dir, 55)
	ctx := context.Background()

	consolidatedResp := buildConsolidatedResponse(5)
	mock := &mockLLMClient{response: consolidatedResp}

	err := fs.ConsolidateWithLLM(ctx, mock, 50)
	if err != nil {
		t.Fatalf("ConsolidateWithLLM failed: %v", err)
	}

	// Verify backup exists.
	bakPath := filepath.Join(dir, "MEMORY.md.bak")
	if _, err := os.Stat(bakPath); err != nil {
		t.Fatalf("backup file should exist after consolidation: %v", err)
	}

	// Verify consolidated entries.
	entries, err := fs.Read(ctx)
	if err != nil {
		t.Fatalf("Read after consolidation failed: %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("expected 5 consolidated entries, got %d", len(entries))
	}
}

func TestConsolidateWithLLMBackupCreated(t *testing.T) {
	dir := t.TempDir()
	fs := writeNEntries(t, dir, 55)
	ctx := context.Background()

	// Read original content before consolidation.
	originalData, err := os.ReadFile(filepath.Join(dir, "MEMORY.md"))
	if err != nil {
		t.Fatalf("read original: %v", err)
	}

	consolidatedResp := buildConsolidatedResponse(3)
	mock := &mockLLMClient{response: consolidatedResp}

	err = fs.ConsolidateWithLLM(ctx, mock, 50)
	if err != nil {
		t.Fatalf("ConsolidateWithLLM failed: %v", err)
	}

	// Verify backup content matches original.
	bakPath := filepath.Join(dir, "MEMORY.md.bak")
	bakData, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}

	if string(bakData) != string(originalData) {
		t.Fatalf("backup content does not match original")
	}
}

func TestConsolidateWithLLMContentReplaced(t *testing.T) {
	dir := t.TempDir()
	fs := writeNEntries(t, dir, 55)
	ctx := context.Background()

	// Count entries before consolidation.
	countBefore, err := fs.Count()
	if err != nil {
		t.Fatalf("Count before failed: %v", err)
	}
	if countBefore != 55 {
		t.Fatalf("expected 55 entries before consolidation, got %d", countBefore)
	}

	consolidatedResp := buildConsolidatedResponse(10)
	mock := &mockLLMClient{response: consolidatedResp}

	err = fs.ConsolidateWithLLM(ctx, mock, 50)
	if err != nil {
		t.Fatalf("ConsolidateWithLLM failed: %v", err)
	}

	// Count entries after — should be fewer.
	countAfter, err := fs.Count()
	if err != nil {
		t.Fatalf("Count after failed: %v", err)
	}
	if countAfter >= countBefore {
		t.Fatalf("expected fewer entries after consolidation: before=%d, after=%d", countBefore, countAfter)
	}
	if countAfter != 10 {
		t.Fatalf("expected 10 consolidated entries, got %d", countAfter)
	}
}

func TestConsolidateWithLLMLLMError(t *testing.T) {
	dir := t.TempDir()
	fs := writeNEntries(t, dir, 55)
	ctx := context.Background()

	// Read original content.
	originalData, err := os.ReadFile(filepath.Join(dir, "MEMORY.md"))
	if err != nil {
		t.Fatalf("read original: %v", err)
	}

	mock := &mockLLMClient{err: fmt.Errorf("LLM service unavailable")}

	err = fs.ConsolidateWithLLM(ctx, mock, 50)
	if err == nil {
		t.Fatalf("expected error from ConsolidateWithLLM when LLM fails")
	}
	if !strings.Contains(err.Error(), "LLM service unavailable") {
		t.Fatalf("expected error to contain LLM error message, got: %v", err)
	}

	// Verify original file is unchanged.
	currentData, err := os.ReadFile(filepath.Join(dir, "MEMORY.md"))
	if err != nil {
		t.Fatalf("read current: %v", err)
	}
	if string(currentData) != string(originalData) {
		t.Fatalf("original file should be unchanged after LLM error")
	}
}

func TestFormatEntriesForLLM(t *testing.T) {
	entries := []Entry{
		{
			Timestamp: time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC),
			Content:   "First entry content.",
			Tags:      []string{"tag1", "tag2"},
		},
		{
			Timestamp: time.Date(2025, 6, 1, 11, 0, 0, 0, time.UTC),
			Content:   "Second entry content.",
			Tags:      nil,
		},
		{
			Timestamp: time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
			Content:   "Third entry content.",
			Tags:      []string{"important"},
		},
	}

	result := formatEntriesForLLM(entries)

	// Verify it contains all timestamps.
	if !strings.Contains(result, "## 2025-06-01T10:00:00Z") {
		t.Error("missing first entry timestamp")
	}
	if !strings.Contains(result, "## 2025-06-01T11:00:00Z") {
		t.Error("missing second entry timestamp")
	}
	if !strings.Contains(result, "## 2025-06-01T12:00:00Z") {
		t.Error("missing third entry timestamp")
	}

	// Verify it contains all content.
	if !strings.Contains(result, "First entry content.") {
		t.Error("missing first entry content")
	}
	if !strings.Contains(result, "Second entry content.") {
		t.Error("missing second entry content")
	}
	if !strings.Contains(result, "Third entry content.") {
		t.Error("missing third entry content")
	}

	// Verify tags are present for entries that have them.
	if !strings.Contains(result, "**Tags**: tag1, tag2") {
		t.Error("missing first entry tags")
	}
	if !strings.Contains(result, "**Tags**: important") {
		t.Error("missing third entry tags")
	}

	// Verify the output is valid markdown that can be parsed back.
	parsed, err := parseMarkdown([]byte(result))
	if err != nil {
		t.Fatalf("formatted output should be valid markdown, parse error: %v", err)
	}
	if len(parsed) != len(entries) {
		t.Fatalf("parsed %d entries from formatted output, want %d", len(parsed), len(entries))
	}
}

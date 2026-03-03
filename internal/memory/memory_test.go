package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestReadWrite(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStore(filepath.Join(dir, "MEMORY.md"))
	ctx := context.Background()

	entries := []Entry{
		{
			Timestamp: time.Date(2025, 2, 28, 10, 0, 0, 0, time.UTC),
			Content:   "First entry: completed auth module refactoring.",
			Tags:      []string{"task", "auth"},
		},
		{
			Timestamp: time.Date(2025, 2, 28, 11, 0, 0, 0, time.UTC),
			Content:   "Second entry: fixed race condition in SSE handler.",
			Tags:      []string{"debug", "fix"},
		},
		{
			Timestamp: time.Date(2025, 2, 28, 12, 0, 0, 0, time.UTC),
			Content:   "Third entry: deployed to staging environment.",
			Tags:      []string{"deploy"},
		},
	}

	// Write all entries.
	for _, e := range entries {
		if err := fs.Write(ctx, e); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	// Read them back.
	got, err := fs.Read(ctx)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(got) != len(entries) {
		t.Fatalf("got %d entries, want %d", len(got), len(entries))
	}

	for i, e := range entries {
		if !got[i].Timestamp.Equal(e.Timestamp) {
			t.Errorf("entry %d: timestamp = %v, want %v", i, got[i].Timestamp, e.Timestamp)
		}
		if got[i].Content != e.Content {
			t.Errorf("entry %d: content = %q, want %q", i, got[i].Content, e.Content)
		}
		if len(got[i].Tags) != len(e.Tags) {
			t.Errorf("entry %d: tags count = %d, want %d", i, len(got[i].Tags), len(e.Tags))
		} else {
			for j := range e.Tags {
				if got[i].Tags[j] != e.Tags[j] {
					t.Errorf("entry %d tag %d: got %q, want %q", i, j, got[i].Tags[j], e.Tags[j])
				}
			}
		}
	}
}

func TestSearch(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStore(filepath.Join(dir, "MEMORY.md"))
	ctx := context.Background()

	entries := []Entry{
		{
			Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			Content:   "Implemented authentication with JWT tokens.",
			Tags:      []string{"auth", "security"},
		},
		{
			Timestamp: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			Content:   "Fixed database connection pooling issue.",
			Tags:      []string{"database", "fix"},
		},
		{
			Timestamp: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
			Content:   "Added rate limiting to auth endpoints.",
			Tags:      []string{"auth", "rate-limit"},
		},
	}

	for _, e := range entries {
		if err := fs.Write(ctx, e); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	// Search by content keyword.
	results, err := fs.Search(ctx, "auth")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("search 'auth': got %d results, want 2", len(results))
	}

	// Search by tag.
	results, err = fs.Search(ctx, "database")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("search 'database': got %d results, want 1", len(results))
	}

	// Search with no matches.
	results, err = fs.Search(ctx, "kubernetes")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("search 'kubernetes': got %d results, want 0", len(results))
	}

	// Case-insensitive search.
	results, err = fs.Search(ctx, "JWT")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("search 'JWT': got %d results, want 1", len(results))
	}
}

func TestCount(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStore(filepath.Join(dir, "MEMORY.md"))
	ctx := context.Background()

	// Empty file.
	count, err := fs.Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("empty count = %d, want 0", count)
	}

	// Write N entries.
	n := 5
	for i := 0; i < n; i++ {
		err := fs.Write(ctx, Entry{
			Timestamp: time.Date(2025, 3, 1, i, 0, 0, 0, time.UTC),
			Content:   fmt.Sprintf("Entry number %d", i+1),
			Tags:      []string{"test"},
		})
		if err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}
	}

	count, err = fs.Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != n {
		t.Fatalf("count = %d, want %d", count, n)
	}
}

func TestEmptyFile(t *testing.T) {
	dir := t.TempDir()
	// Point to a file that does not exist.
	fs := NewFileStore(filepath.Join(dir, "nonexistent", "MEMORY.md"))
	ctx := context.Background()

	entries, err := fs.Read(ctx)
	if err != nil {
		t.Fatalf("Read on non-existent file should not error, got: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}

	// Count should also return 0.
	count, err := fs.Count()
	if err != nil {
		t.Fatalf("Count on non-existent file should not error, got: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected count 0, got %d", count)
	}

	// Search should return empty.
	results, err := fs.Search(ctx, "anything")
	if err != nil {
		t.Fatalf("Search on non-existent file should not error, got: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 search results, got %d", len(results))
	}
}

func TestConcurrent(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStore(filepath.Join(dir, "MEMORY.md"))
	ctx := context.Background()

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	errs := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			err := fs.Write(ctx, Entry{
				Timestamp: time.Date(2025, 4, 1, idx, 0, 0, 0, time.UTC),
				Content:   fmt.Sprintf("Concurrent entry %d", idx),
				Tags:      []string{"concurrent", fmt.Sprintf("g%d", idx)},
			})
			if err != nil {
				errs <- fmt.Errorf("goroutine %d: %w", idx, err)
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("Concurrent write error: %v", err)
	}

	// Verify all entries were written.
	entries, err := fs.Read(ctx)
	if err != nil {
		t.Fatalf("Read after concurrent writes failed: %v", err)
	}

	if len(entries) != goroutines {
		t.Fatalf("got %d entries, want %d (some writes lost)", len(entries), goroutines)
	}

	// Verify no corruption: each entry should parse correctly.
	for i, e := range entries {
		if e.Content == "" {
			t.Errorf("entry %d has empty content", i)
		}
		if e.Timestamp.IsZero() {
			t.Errorf("entry %d has zero timestamp", i)
		}
	}
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	memPath := filepath.Join(dir, "MEMORY.md")
	fs := NewFileStore(memPath)
	ctx := context.Background()

	err := fs.Write(ctx, Entry{
		Timestamp: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
		Content:   "Atomic write test entry.",
		Tags:      []string{"atomic"},
	})
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify the main file exists.
	if _, err := os.Stat(memPath); err != nil {
		t.Fatalf("MEMORY.md should exist: %v", err)
	}

	// Verify the temp file does not linger.
	tmpPath := filepath.Join(dir, ".MEMORY.md.tmp")
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Fatalf(".MEMORY.md.tmp should not exist after write, got err: %v", err)
	}
}

func TestParseMarkdown(t *testing.T) {
	t.Run("with_tags", func(t *testing.T) {
		input := `## 2025-02-28T10:30:00Z
**Tags**: task, opencode, success
Completed refactoring of auth module. User was satisfied with the result.

## 2025-02-28T11:00:00Z
**Tags**: debug, error
Found a race condition in the SSE handler. Fixed with sync.Mutex.
`
		entries, err := parseMarkdown([]byte(input))
		if err != nil {
			t.Fatalf("parseMarkdown failed: %v", err)
		}

		if len(entries) != 2 {
			t.Fatalf("got %d entries, want 2", len(entries))
		}

		// First entry.
		e := entries[0]
		wantTS := time.Date(2025, 2, 28, 10, 30, 0, 0, time.UTC)
		if !e.Timestamp.Equal(wantTS) {
			t.Errorf("entry 0 timestamp = %v, want %v", e.Timestamp, wantTS)
		}
		if len(e.Tags) != 3 || e.Tags[0] != "task" || e.Tags[1] != "opencode" || e.Tags[2] != "success" {
			t.Errorf("entry 0 tags = %v, want [task opencode success]", e.Tags)
		}
		wantContent := "Completed refactoring of auth module. User was satisfied with the result."
		if e.Content != wantContent {
			t.Errorf("entry 0 content = %q, want %q", e.Content, wantContent)
		}

		// Second entry.
		e = entries[1]
		wantTS = time.Date(2025, 2, 28, 11, 0, 0, 0, time.UTC)
		if !e.Timestamp.Equal(wantTS) {
			t.Errorf("entry 1 timestamp = %v, want %v", e.Timestamp, wantTS)
		}
		if len(e.Tags) != 2 || e.Tags[0] != "debug" || e.Tags[1] != "error" {
			t.Errorf("entry 1 tags = %v, want [debug error]", e.Tags)
		}
	})

	t.Run("without_tags", func(t *testing.T) {
		input := `## 2025-03-01T09:00:00Z
A simple entry without any tags.
`
		entries, err := parseMarkdown([]byte(input))
		if err != nil {
			t.Fatalf("parseMarkdown failed: %v", err)
		}

		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}

		e := entries[0]
		if len(e.Tags) != 0 {
			t.Errorf("expected no tags, got %v", e.Tags)
		}
		if e.Content != "A simple entry without any tags." {
			t.Errorf("content = %q, want %q", e.Content, "A simple entry without any tags.")
		}
	})

	t.Run("empty_input", func(t *testing.T) {
		entries, err := parseMarkdown([]byte(""))
		if err != nil {
			t.Fatalf("parseMarkdown failed: %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("got %d entries, want 0", len(entries))
		}
	})

	t.Run("whitespace_only", func(t *testing.T) {
		entries, err := parseMarkdown([]byte("  \n\n  \n"))
		if err != nil {
			t.Fatalf("parseMarkdown failed: %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("got %d entries, want 0", len(entries))
		}
	})
}

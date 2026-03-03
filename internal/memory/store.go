// Package memory provides persistent memory storage using Markdown files.
// It implements append-only memory with atomic writes and substring search.
package memory

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/startower-observability/blackcat/internal/types"
)

// Entry is a single memory entry.
// Defined locally to avoid dependency timing issues with the types package.
// TODO: Consider using types.Entry once the types package is stable.
type Entry struct {
	Timestamp time.Time
	Content   string
	Tags      []string
}

// FileStore implements persistent memory storage using a Markdown file.
// It uses atomic file writes (write-to-temp + rename) and a RWMutex for
// thread safety.
type FileStore struct {
	path string
	mu   sync.RWMutex
}

// NewFileStore creates a new FileStore that reads/writes to the given path.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

// Read parses all entries from the MEMORY.md file.
// Returns an empty slice (not an error) if the file does not exist.
func (fs *FileStore) Read(ctx context.Context) ([]Entry, error) {
	fs.mu.RLock()
	data, err := os.ReadFile(fs.path)
	fs.mu.RUnlock()

	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("memory: read file: %w", err)
	}

	return parseMarkdown(data)
}

// Write appends an entry to the MEMORY.md file using atomic write.
func (fs *FileStore) Write(ctx context.Context, entry Entry) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Read existing content.
	existing, err := os.ReadFile(fs.path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("memory: read existing: %w", err)
	}

	// Format the new entry.
	newEntry := formatEntry(entry)

	// Build full content.
	var buf bytes.Buffer
	if len(existing) > 0 {
		buf.Write(existing)
		// Ensure there's a blank line before the new entry.
		if !bytes.HasSuffix(existing, []byte("\n\n")) {
			if bytes.HasSuffix(existing, []byte("\n")) {
				buf.WriteByte('\n')
			} else {
				buf.WriteString("\n\n")
			}
		}
	}
	buf.WriteString(newEntry)

	// Atomic write: write to temp file, then rename.
	dir := filepath.Dir(fs.path)
	base := filepath.Base(fs.path)
	tmpPath := filepath.Join(dir, "."+base+".tmp")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("memory: create dir: %w", err)
	}

	if err := os.WriteFile(tmpPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("memory: write temp: %w", err)
	}

	if err := os.Rename(tmpPath, fs.path); err != nil {
		// Clean up temp file on rename failure.
		os.Remove(tmpPath)
		return fmt.Errorf("memory: rename: %w", err)
	}

	return nil
}

// Search returns all entries whose content or tags contain the query substring.
func (fs *FileStore) Search(ctx context.Context, query string) ([]Entry, error) {
	entries, err := fs.Read(ctx)
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	var results []Entry
	for _, e := range entries {
		if matchesQuery(e, queryLower) {
			results = append(results, e)
		}
	}
	return results, nil
}

// Count returns the number of entries in the memory file.
func (fs *FileStore) Count() (int, error) {
	fs.mu.RLock()
	data, err := os.ReadFile(fs.path)
	fs.mu.RUnlock()

	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("memory: count read: %w", err)
	}

	entries, err := parseMarkdown(data)
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}

// Consolidate is a no-op without an LLM client. Use ConsolidateWithLLM instead.
func (fs *FileStore) Consolidate(ctx context.Context) error {
	// Still a no-op without LLM client - use ConsolidateWithLLM instead
	return nil
}

// formatEntry formats a single Entry as a Markdown section.
func formatEntry(e Entry) string {
	var buf strings.Builder
	buf.WriteString("## ")
	buf.WriteString(e.Timestamp.UTC().Format(time.RFC3339))
	buf.WriteByte('\n')

	if len(e.Tags) > 0 {
		buf.WriteString("**Tags**: ")
		buf.WriteString(strings.Join(e.Tags, ", "))
		buf.WriteByte('\n')
	}

	buf.WriteString(strings.TrimSpace(e.Content))
	buf.WriteByte('\n')

	return buf.String()
}

// parseMarkdown parses the MEMORY.md content into Entry slices.
func parseMarkdown(data []byte) ([]Entry, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return []Entry{}, nil
	}

	var entries []Entry
	scanner := bufio.NewScanner(bytes.NewReader(data))

	var current *Entry
	var contentLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "## ") {
			// Save the previous entry.
			if current != nil {
				current.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))
				entries = append(entries, *current)
			}

			// Parse timestamp from heading.
			tsStr := strings.TrimPrefix(line, "## ")
			ts, err := time.Parse(time.RFC3339, strings.TrimSpace(tsStr))
			if err != nil {
				return nil, fmt.Errorf("memory: parse timestamp %q: %w", tsStr, err)
			}

			current = &Entry{Timestamp: ts}
			contentLines = nil
			continue
		}

		if current == nil {
			continue
		}

		// Parse tags line.
		if strings.HasPrefix(line, "**Tags**: ") {
			tagStr := strings.TrimPrefix(line, "**Tags**: ")
			tags := strings.Split(tagStr, ", ")
			for i, t := range tags {
				tags[i] = strings.TrimSpace(t)
			}
			current.Tags = tags
			continue
		}

		contentLines = append(contentLines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("memory: scan: %w", err)
	}

	// Save last entry.
	if current != nil {
		current.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))
		entries = append(entries, *current)
	}

	return entries, nil
}

// matchesQuery checks if an entry's content or tags contain the query (case-insensitive).
func matchesQuery(e Entry, queryLower string) bool {
	if strings.Contains(strings.ToLower(e.Content), queryLower) {
		return true
	}
	for _, tag := range e.Tags {
		if strings.Contains(strings.ToLower(tag), queryLower) {
			return true
		}
	}
	return false
}

// formatEntriesForLLM formats all entries into a single Markdown string for LLM consumption.
func formatEntriesForLLM(entries []Entry) string {
	var buf strings.Builder
	for i, e := range entries {
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(formatEntry(e))
	}
	return buf.String()
}

// ConsolidateWithLLM sends all entries to the LLM for consolidation when
// entry count exceeds the given threshold. Creates a backup before replacing.
func (fs *FileStore) ConsolidateWithLLM(ctx context.Context, llmClient types.LLMClient, threshold int) error {
	entries, err := fs.Read(ctx)
	if err != nil {
		return fmt.Errorf("memory: consolidate read: %w", err)
	}

	if len(entries) < threshold {
		return nil
	}

	// Create backup: read current content and write to .bak path.
	fs.mu.RLock()
	original, err := os.ReadFile(fs.path)
	fs.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("memory: consolidate backup read: %w", err)
	}

	bakPath := fs.path + ".bak"
	if err := os.WriteFile(bakPath, original, 0o644); err != nil {
		return fmt.Errorf("memory: consolidate backup write: %w", err)
	}

	// Format entries for the LLM.
	formatted := formatEntriesForLLM(entries)

	// Call LLM for consolidation.
	messages := []types.LLMMessage{
		{
			Role:    "system",
			Content: "You are a memory consolidation assistant. Consolidate the following memory entries into a concise summary. Preserve key facts, decisions, user preferences, and important context. Remove redundant or outdated entries. Output in the same Markdown format with ## timestamp headers.",
		},
		{
			Role:    "user",
			Content: formatted,
		},
	}

	resp, err := llmClient.Chat(ctx, messages, nil)
	if err != nil {
		return fmt.Errorf("memory: consolidate llm: %w", err)
	}

	// Parse the LLM response back into entries.
	_, err = parseMarkdown([]byte(resp.Content))
	if err != nil {
		return fmt.Errorf("memory: consolidate parse response: %w", err)
	}

	// Atomic write: write consolidated content to temp file, then rename.
	fs.mu.Lock()
	defer fs.mu.Unlock()

	dir := filepath.Dir(fs.path)
	base := filepath.Base(fs.path)
	tmpPath := filepath.Join(dir, "."+base+".tmp")

	if err := os.WriteFile(tmpPath, []byte(resp.Content), 0o644); err != nil {
		return fmt.Errorf("memory: consolidate write temp: %w", err)
	}

	if err := os.Rename(tmpPath, fs.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("memory: consolidate rename: %w", err)
	}

	return nil
}

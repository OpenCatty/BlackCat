package memory

import "context"

// Store is the interface for memory storage backends.
// Both FileStore (Markdown-based) and SQLiteStore (SQLite + FTS5) implement this.
type Store interface {
	// Read returns all stored memory entries.
	Read(ctx context.Context) ([]Entry, error)
	// Write appends a new memory entry.
	Write(ctx context.Context, entry Entry) error
	// Search returns entries matching the query string.
	Search(ctx context.Context, query string) ([]Entry, error)
	// Consolidate performs optional cleanup/compaction.
	Consolidate(ctx context.Context) error
}

package memory

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements Store with SQLite and FTS5 full-text search.
type SQLiteStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewSQLiteStore creates a new SQLite-backed memory store at the given path.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("memory: create dir: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("memory: open sqlite: %w", err)
	}

	if err := createSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("memory: create schema: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func createSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS memories (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		content    TEXT    NOT NULL,
		tags       TEXT    DEFAULT '',
		source     TEXT    DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
		content, tags,
		content='memories',
		content_rowid='id'
	);

	CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
		INSERT INTO memories_fts(rowid, content, tags)
		VALUES (new.id, new.content, new.tags);
	END;

	CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
		INSERT INTO memories_fts(memories_fts, rowid, content, tags)
		VALUES ('delete', old.id, old.content, old.tags);
	END;
	`
	_, err := db.Exec(schema)
	return err
}

// Read returns all entries (up to 100, in chronological order).
func (s *SQLiteStore) Read(ctx context.Context) ([]Entry, error) {
	return s.Recent(ctx, 100)
}

// Write appends a new memory entry.
func (s *SQLiteStore) Write(ctx context.Context, entry Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tags := strings.Join(entry.Tags, ",")
	ts := entry.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO memories (content, tags, source, created_at) VALUES (?, ?, ?, ?)`,
		entry.Content, tags, tagSource(entry.Tags), ts.UTC(),
	)
	if err != nil {
		return fmt.Errorf("memory: sqlite write: %w", err)
	}
	return nil
}

// Search returns entries matching the query using FTS5.
func (s *SQLiteStore) Search(ctx context.Context, query string) ([]Entry, error) {
	return s.SearchWithLimit(ctx, query, 10)
}

// SearchWithLimit performs FTS5 search with a configurable result limit.
func (s *SQLiteStore) SearchWithLimit(ctx context.Context, query string, limit int) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT m.content, m.tags, m.created_at
		 FROM memories m
		 JOIN memories_fts fts ON m.id = fts.rowid
		 WHERE memories_fts MATCH ?
		 ORDER BY rank
		 LIMIT ?`,
		ftsQuery(query), limit,
	)
	if err != nil {
		// Fallback to LIKE if FTS fails (e.g. invalid query syntax)
		return s.searchLike(ctx, query, limit)
	}
	defer rows.Close()

	return scanRows(rows)
}

func (s *SQLiteStore) searchLike(ctx context.Context, query string, limit int) ([]Entry, error) {
	like := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx,
		`SELECT content, tags, created_at FROM memories
		 WHERE content LIKE ? OR tags LIKE ?
		 ORDER BY created_at DESC LIMIT ?`,
		like, like, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("memory: sqlite like search: %w", err)
	}
	defer rows.Close()

	return scanRows(rows)
}

// Recent returns the most recent entries in chronological order.
func (s *SQLiteStore) Recent(ctx context.Context, limit int) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT content, tags, created_at FROM memories ORDER BY created_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("memory: sqlite recent: %w", err)
	}
	defer rows.Close()

	entries, err := scanRows(rows)
	if err != nil {
		return nil, err
	}

	// Reverse to chronological order (oldest first)
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, nil
}

// Add is a convenience method for adding a memory entry.
func (s *SQLiteStore) Add(ctx context.Context, content string, tags []string, source string) error {
	return s.Write(ctx, Entry{
		Timestamp: time.Now(),
		Content:   content,
		Tags:      tags,
	})
}

// Count returns the total number of stored memories.
func (s *SQLiteStore) Count(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memories`).Scan(&n); err != nil {
		return 0, fmt.Errorf("memory: sqlite count: %w", err)
	}
	return n, nil
}

// Close closes the SQLite database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Consolidate is a no-op for SQLiteStore.
func (s *SQLiteStore) Consolidate(ctx context.Context) error {
	return nil
}

// MigrateFromFileStore imports entries from a FileStore into SQLite.
func (s *SQLiteStore) MigrateFromFileStore(ctx context.Context, fs *FileStore) (int, error) {
	entries, err := fs.Read(ctx)
	if err != nil {
		return 0, fmt.Errorf("memory: migrate read: %w", err)
	}

	imported := 0
	for _, e := range entries {
		if err := s.Write(ctx, e); err != nil {
			continue
		}
		imported++
	}
	return imported, nil
}

func scanRows(rows *sql.Rows) ([]Entry, error) {
	var entries []Entry
	for rows.Next() {
		var content, tags string
		var createdAt time.Time
		if err := rows.Scan(&content, &tags, &createdAt); err != nil {
			return nil, fmt.Errorf("memory: sqlite scan: %w", err)
		}

		var tagSlice []string
		if tags != "" {
			for _, t := range strings.Split(tags, ",") {
				if s := strings.TrimSpace(t); s != "" {
					tagSlice = append(tagSlice, s)
				}
			}
		}

		entries = append(entries, Entry{
			Timestamp: createdAt,
			Content:   content,
			Tags:      tagSlice,
		})
	}
	return entries, rows.Err()
}

// ftsQuery builds an FTS5 MATCH query from user input.
func ftsQuery(query string) string {
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return `""`
	}
	escaped := make([]string, len(terms))
	for i, t := range terms {
		t = strings.NewReplacer(`"`, "", "*", "", "(", "", ")", "", ":", "").Replace(t)
		escaped[i] = `"` + t + `"` + "*"
	}
	return strings.Join(escaped, " ")
}

// tagSource extracts the first tag as source, or returns empty string.
func tagSource(tags []string) string {
	if len(tags) > 0 {
		return tags[0]
	}
	return ""
}

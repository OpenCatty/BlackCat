package taskqueue

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// createSchema initialises the tasks table and indices.
func createSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		task_type    TEXT    NOT NULL,
		status       TEXT    NOT NULL DEFAULT 'pending',
		payload      TEXT    NOT NULL DEFAULT '',
		result       TEXT    NOT NULL DEFAULT '',
		error        TEXT    NOT NULL DEFAULT '',
		recipient_id TEXT    NOT NULL DEFAULT '',
		retry_count  INTEGER NOT NULL DEFAULT 0,
		max_retries  INTEGER NOT NULL DEFAULT 3,
		timeout_secs INTEGER NOT NULL DEFAULT 1800,
		created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME
	);

	CREATE INDEX IF NOT EXISTS idx_tasks_status     ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_type       ON tasks(task_type);
	CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);
	`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("taskqueue: create schema: %w", err)
	}
	return nil
}

// openDB opens (or creates) the SQLite database at dbPath.
func openDB(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("taskqueue: create dir: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("taskqueue: open sqlite: %w", err)
	}

	if err := createSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// cleanupOldTasks deletes completed/failed tasks older than 7 days.
func cleanupOldTasks(db *sql.DB) {
	res, err := db.Exec(
		`DELETE FROM tasks
		 WHERE status IN (?, ?)
		   AND completed_at IS NOT NULL
		   AND completed_at < datetime('now', '-7 days')`,
		StatusCompleted, StatusFailed,
	)
	if err != nil {
		slog.Warn("taskqueue: cleanup failed", "error", err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		slog.Info("taskqueue: cleaned up old tasks", "deleted", n)
	}
}
